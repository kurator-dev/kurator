/*
Copyright Kurator Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
	http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package fleet

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/robfig/cron/v3"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupapi "kurator.dev/kurator/pkg/apis/backups/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	"kurator.dev/kurator/pkg/fleet-manager/plugin"
)

// fetchDestinationClusters retrieves the clusters from the specified destination and filters them based on the 'Clusters' defined in the destination. It returns a map of selected clusters along with any error encountered during the process.
func (b *BackupManager) fetchDestinationClusters(ctx context.Context, namespace string, destination backupapi.Destination) (map[ClusterKey]*fleetCluster, error) {
	// Fetch fleet instance
	fleet := &fleetapi.Fleet{}
	fleetKey := client.ObjectKey{
		Namespace: namespace,
		Name:      destination.Fleet,
	}
	if err := b.Client.Get(ctx, fleetKey, fleet); err != nil {
		return nil, fmt.Errorf("failed to retrieve fleet instance '%s' in namespace '%s': %w", destination.Fleet, namespace, err)
	}

	fleetClusters, err := buildFleetClusters(ctx, b.Client, fleet)
	if err != nil {
		return nil, fmt.Errorf("failed to build fleet clusters from fleet instance '%s': %w", fleet.Name, err)
	}

	// If no destination.Clusters defined, return all clusters
	if len(destination.Clusters) == 0 {
		return fleetClusters, nil
	}

	selectedFleetCluster := make(map[ClusterKey]*fleetCluster)
	// Check and return clusters with client
	for _, cluster := range destination.Clusters {
		name := cluster.Name
		kind := cluster.Kind
		found := false
		for key, f := range fleetClusters {
			if key.Name == name && key.Kind == kind {
				found = true
				selectedFleetCluster[key] = f
			}
		}
		if !found {
			return nil, fmt.Errorf("current clusters: clustername: %s, clusterKind: %s in destination is not recorede in the fleet: %s", name, kind, fleet.Name)
		}
	}
	return selectedFleetCluster, nil
}

// fetchRestoreDestinationClusters retrieves the destination clusters for a restore operation.
// It first fetches the referred backup and then determines the destination clusters based on the restore and backup specifications:
// If the restore destination is not set, it returns the clusters from the backup destination.
// If set, it ensures that the restore destination is a subset of the backup destination.
//
// Returns:
// - string: The name of the fleet where the restore's set of fleetClusters resides.
// - map[ClusterKey]*fleetCluster: A map of cluster keys to fleet clusters.
// - error: An error object indicating any issues encountered during the operation.
func (b *BackupManager) fetchRestoreDestinationClusters(ctx context.Context, restore *backupapi.Restore) (string, map[ClusterKey]*fleetCluster, error) {
	// Retrieve the referred backup in the current Kurator host cluster
	key := client.ObjectKey{
		Name:      restore.Spec.BackupName,
		Namespace: restore.Namespace,
	}
	referredBackup := &backupapi.Backup{}
	if err := b.Client.Get(ctx, key, referredBackup); err != nil {
		return "", nil, fmt.Errorf("failed to retrieve the referred backup '%s': %w", restore.Spec.BackupName, err)
	}

	// Get the base clusters from the referred backup
	baseClusters, err := b.fetchDestinationClusters(ctx, restore.Namespace, referredBackup.Spec.Destination)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch fleet clusters for the backup '%s': %w", referredBackup.Name, err)
	}

	// If the restore destination is not set, return the base fleet clusters directly
	if restore.Spec.Destination == nil {
		return referredBackup.Spec.Destination.Fleet, baseClusters, nil
	}

	// If the restore destination is set, try to get the clusters from the restore destination
	restoreClusters, err := b.fetchDestinationClusters(ctx, restore.Namespace, *restore.Spec.Destination)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch restore clusters for the restore '%s': %w", restore.Name, err)
	}

	// Check the fleet and clusters between the restore and the referred backup
	if referredBackup.Spec.Destination.Fleet != restore.Spec.Destination.Fleet {
		// if we make sure only one fleet in one ns, this error will never happen
		return "", nil, errors.New("the restore destination fleet must be the same as the backup's")
	}

	if !isFleetClusterSubset(baseClusters, restoreClusters) {
		return "", nil, errors.New("the restore clusters must be a subset of the base clusters")
	}

	return restore.Spec.Destination.Fleet, restoreClusters, nil
}

// isFleetClusterSubset is the helper function to check if one set of clusters is a subset of another
func isFleetClusterSubset(baseClusters, subsetClusters map[ClusterKey]*fleetCluster) bool {
	for key := range subsetClusters {
		if _, exists := baseClusters[key]; !exists {
			return false
		}
	}
	return true
}

func (b *BackupManager) fetchMigrateSourceClusters(ctx context.Context, namespace string, destination backupapi.Destination) (sourceClusterName, sourceClusterNameKind string, sourceCluster *fleetCluster, err error) {
	fleetClusters, err := b.fetchDestinationClusters(ctx, namespace, destination)
	if err != nil {
		return "", "", nil, err
	}

	// SourceCluster must contain one Clusters, it is ensured by admission webhook
	for key, value := range fleetClusters {
		sourceClusterName = key.Name
		sourceClusterNameKind = key.Kind
		sourceCluster = value
	}

	return sourceClusterName, sourceClusterNameKind, sourceCluster, err
}

// buildVeleroBackupInstance constructs a Velero Backup instance configured to perform a backup operation on the specified cluster.
func buildVeleroBackupInstance(backupSpec *backupapi.BackupSpec, labels map[string]string, veleroBackupName string) *velerov1.Backup {
	veleroBackup := &velerov1.Backup{
		ObjectMeta: generateVeleroResourceObjectMeta(veleroBackupName, labels),
		Spec:       buildVeleroBackupSpec(backupSpec.Policy),
	}
	return veleroBackup
}

// buildVeleroBackupInstanceUsingMigrate constructs a Velero Backup instance configured to perform a backup operation on the specified cluster using migrate config.
func buildVeleroBackupInstanceUsingMigrate(migrateSpec *backupapi.MigrateSpec, labels map[string]string, veleroBackupName string) *velerov1.Backup {
	// when using velero backup, only care about migrateSpec.Policy.ResourceFilter
	backupParam := &backupapi.BackupSpec{}
	if migrateSpec.Policy != nil {
		backupParam.Policy = &backupapi.BackupPolicy{}
		backupParam.Policy.ResourceFilter = migrateSpec.Policy.ResourceFilter
	}
	return buildVeleroBackupInstance(backupParam, labels, veleroBackupName)
}

// buildVeleroScheduleInstance constructs a Velero Schedule instance configured to schedule backup operations on the specified cluster.
func buildVeleroScheduleInstance(backupSpec *backupapi.BackupSpec, labels map[string]string, veleroBackupName string) *velerov1.Schedule {
	veleroSchedule := &velerov1.Schedule{
		ObjectMeta: generateVeleroResourceObjectMeta(veleroBackupName, labels),
		Spec: velerov1.ScheduleSpec{
			Template: buildVeleroBackupSpec(backupSpec.Policy),
			Schedule: backupSpec.Schedule,
		},
	}
	return veleroSchedule
}

// buildVeleroScheduleInstance constructs a Velero Restore instance configured to Restore operations on the specified cluster.
func buildVeleroRestoreInstance(restoreSpec *backupapi.RestoreSpec, labels map[string]string, veleroBackupName, veleroRestoreName string) *velerov1.Restore {
	veleroRestore := &velerov1.Restore{
		ObjectMeta: generateVeleroResourceObjectMeta(veleroRestoreName, labels),
		Spec: velerov1.RestoreSpec{
			BackupName: veleroBackupName,
		},
	}
	if restoreSpec.Policy != nil {
		veleroRestore.Spec.NamespaceMapping = restoreSpec.Policy.NamespaceMapping
		veleroRestore.Spec.PreserveNodePorts = restoreSpec.Policy.PreserveNodePorts
		// restore will not contain namespace scope filter
		if restoreSpec.Policy.ResourceFilter != nil {
			veleroRestore.Spec.IncludedNamespaces = restoreSpec.Policy.ResourceFilter.IncludedNamespaces
			veleroRestore.Spec.ExcludedNamespaces = restoreSpec.Policy.ResourceFilter.ExcludedNamespaces
			veleroRestore.Spec.IncludedResources = restoreSpec.Policy.ResourceFilter.IncludedResources
			veleroRestore.Spec.ExcludedResources = restoreSpec.Policy.ResourceFilter.ExcludedResources
			veleroRestore.Spec.IncludeClusterResources = restoreSpec.Policy.ResourceFilter.IncludeClusterResources
			veleroRestore.Spec.LabelSelector = restoreSpec.Policy.ResourceFilter.LabelSelector
			veleroRestore.Spec.OrLabelSelectors = restoreSpec.Policy.ResourceFilter.OrLabelSelectors
		}
		if restoreSpec.Policy.PreserveStatus != nil {
			veleroRestore.Spec.RestoreStatus = &velerov1.RestoreStatusSpec{
				IncludedResources: restoreSpec.Policy.PreserveStatus.IncludedResources,
				ExcludedResources: restoreSpec.Policy.PreserveStatus.ExcludedResources,
			}
		}
	}

	return veleroRestore
}

// buildVeleroRestoreInstance constructs a Velero Backup instance configured to perform a backup operation on the specified cluster using migrate config.
func buildVeleroRestoreInstanceUsingMigrate(migrateSpec *backupapi.MigrateSpec, labels map[string]string, veleroBackupName, veleroRestoreName string) *velerov1.Restore {
	restoreParam := &backupapi.RestoreSpec{}
	if migrateSpec.Policy != nil {
		restoreParam.Policy = &backupapi.RestorePolicy{
			NamespaceMapping:  migrateSpec.Policy.NamespaceMapping,
			PreserveNodePorts: migrateSpec.Policy.PreserveNodePorts,
			PreserveStatus:    migrateSpec.Policy.MigrateStatus,
		}
	}
	return buildVeleroRestoreInstance(restoreParam, labels, veleroBackupName, veleroRestoreName)
}

func buildVeleroBackupSpec(backupPolicy *backupapi.BackupPolicy) velerov1.BackupSpec {
	if backupPolicy == nil {
		return velerov1.BackupSpec{}
	}
	return velerov1.BackupSpec{
		TTL:                              backupPolicy.TTL,
		OrderedResources:                 backupPolicy.OrderedResources,
		IncludedNamespaces:               backupPolicy.ResourceFilter.IncludedNamespaces,
		ExcludedNamespaces:               backupPolicy.ResourceFilter.ExcludedNamespaces,
		IncludedResources:                backupPolicy.ResourceFilter.IncludedResources,
		ExcludedResources:                backupPolicy.ResourceFilter.ExcludedResources,
		IncludeClusterResources:          backupPolicy.ResourceFilter.IncludeClusterResources,
		IncludedClusterScopedResources:   backupPolicy.ResourceFilter.IncludedClusterScopedResources,
		ExcludedClusterScopedResources:   backupPolicy.ResourceFilter.ExcludedClusterScopedResources,
		IncludedNamespaceScopedResources: backupPolicy.ResourceFilter.IncludedNamespaceScopedResources,
		ExcludedNamespaceScopedResources: backupPolicy.ResourceFilter.ExcludedNamespaceScopedResources,
		LabelSelector:                    backupPolicy.ResourceFilter.LabelSelector,
		OrLabelSelectors:                 backupPolicy.ResourceFilter.OrLabelSelectors,
	}
}

// buildVeleroBackupInstanceForTest is a helper function for testing for buildVeleroBackupInstance, which constructs a Velero Backup instance with a specified TypeMeta.
func buildVeleroBackupInstanceForTest(backupSpec *backupapi.BackupSpec, labels map[string]string, veleroBackupName string, typeMeta *metav1.TypeMeta) *velerov1.Backup {
	veleroBackup := buildVeleroBackupInstance(backupSpec, labels, veleroBackupName)
	veleroBackup.TypeMeta = *typeMeta // set TypeMeta for test
	return veleroBackup
}

// buildVeleroScheduleInstanceForTest is a helper function for testing buildVeleroScheduleInstance, which constructs a Velero Schedule instance with a specified TypeMeta.
func buildVeleroScheduleInstanceForTest(backupSpec *backupapi.BackupSpec, labels map[string]string, veleroBackupName string, typeMeta *metav1.TypeMeta) *velerov1.Schedule {
	veleroSchedule := buildVeleroScheduleInstance(backupSpec, labels, veleroBackupName)
	veleroSchedule.TypeMeta = *typeMeta
	return veleroSchedule
}

// buildVeleroRestoreInstanceForTest is a helper function for testing buildVeleroScheduleInstance, which constructs a Velero Restore instance with a specified TypeMeta.
func buildVeleroRestoreInstanceForTest(restoreSpec *backupapi.RestoreSpec, labels map[string]string, veleroBackupName, veleroRestoreName string, typeMeta *metav1.TypeMeta) *velerov1.Restore {
	veleroRestore := buildVeleroRestoreInstance(restoreSpec, labels, veleroBackupName, veleroRestoreName)
	veleroRestore.TypeMeta = *typeMeta
	return veleroRestore
}

func createVeleroBackupInstance(cluster *fleetCluster, veleroBackup *velerov1.Backup) error {
	// Get the kubeclient.Interface instance
	kubeClient := cluster.client.VeleroClient()

	// Get the namespace of the Backups
	namespace := veleroBackup.Namespace

	// Create the new Backups
	if _, err := kubeClient.VeleroV1().Backups(namespace).Create(context.TODO(), veleroBackup, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func createVeleroScheduleInstance(cluster *fleetCluster, veleroSchedule *velerov1.Schedule) error {
	// Get the kubeclient.Interface instance
	kubeClient := cluster.client.VeleroClient()

	// Get the namespace of the Schedules
	namespace := veleroSchedule.Namespace

	// Create the new Schedules
	if _, err := kubeClient.VeleroV1().Schedules(namespace).Create(context.TODO(), veleroSchedule, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func createVeleroRestoreInstance(cluster *fleetCluster, veleroRestore *velerov1.Restore) error {
	// Get the kubeclient.Interface instance
	kubeClient := cluster.client.VeleroClient()

	// Get the namespace of the veleroRestore
	namespace := veleroRestore.Namespace

	// Create the new veleroRestore
	if _, err := kubeClient.VeleroV1().Restores(namespace).Create(context.TODO(), veleroRestore, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// allBackupsCompleted checks if all Velero backup statuses in the cluster have been collected and verifies if they have successfully completed.
// If every BackupStatusInCluster.Phase is marked as completed, it indicates that all backups have been successfully completed.
func allBackupsCompleted(status backupapi.BackupStatus) bool {
	for _, detail := range status.Details {
		if detail.BackupStatusInCluster == nil || detail.BackupStatusInCluster.Phase != velerov1.BackupPhaseCompleted {
			return false
		}
	}
	return true
}

// allRestoreCompleted checks if all restore operations are completed by inspecting the phase of each RestoreDetails instance in the provided slice.
func allRestoreCompleted(clusterDetails []*backupapi.RestoreDetails) bool {
	for _, detail := range clusterDetails {
		if detail.RestoreStatusInCluster == nil || detail.RestoreStatusInCluster.Phase != velerov1.RestorePhaseCompleted {
			return false
		}
	}
	return true
}

// deleteCRDInstances deletes instances of various CRDs based on the specified label key and value.
// It iterates over all destination clusters and performs the deletion for the specified resource type.
// Parameters:
// - ctx: context to carry out the API requests
// - namespace: the namespace where the CRDs are located
// - labelKey: the key of the label to filter the CRDs
// - labelValue: the value of the label to filter the CRDs
// - destinationClusters: a map containing information about the destination clusters
// - resourceType: the type of the resource to be deleted (e.g., VeleroBackupKind, VeleroScheduleKind, VeleroRestoreKind)
// Returns:
// - error: if any error occurs during the deletion process
func deleteCRDInstances(ctx context.Context, labelKey string, labelValue string, destinationClusters map[ClusterKey]*fleetCluster, resourceType string) error {
	// Create a label selector to find all Velero backups related to the current backup
	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			labelKey: labelValue,
		},
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	// Iterate over each destination cluster
	for _, cluster := range destinationClusters {
		var listErr error
		var deleteErr error

		veleroClient := cluster.client.VeleroClient().VeleroV1()
		switch resourceType {
		case VeleroBackupKind:
			var backupList *velerov1.BackupList
			if backupList, listErr = veleroClient.Backups(VeleroNamespace).List(ctx, listOptions); listErr != nil {
				return listErr
			}
			if len(backupList.Items) == 0 {
				// All selected items are already deleted
				return nil
			}
			// Delete all backups
			for _, backup := range backupList.Items {
				deleteErr = veleroClient.Backups(VeleroNamespace).Delete(ctx, backup.GetName(), metav1.DeleteOptions{})
				if deleteErr != nil && !apierrors.IsNotFound(deleteErr) {
					return deleteErr
				}
			}
		case VeleroScheduleKind:
			var scheduleList *velerov1.ScheduleList
			if scheduleList, listErr = veleroClient.Schedules(VeleroNamespace).List(ctx, listOptions); listErr != nil {
				return listErr
			}
			if len(scheduleList.Items) == 0 {
				// All selected items are already deleted
				return nil
			}
			// Delete all schedules
			for _, schedule := range scheduleList.Items {
				deleteErr = veleroClient.Schedules(VeleroNamespace).Delete(ctx, schedule.GetName(), metav1.DeleteOptions{})
				if deleteErr != nil && !apierrors.IsNotFound(deleteErr) {
					return deleteErr
				}
			}
		case VeleroRestoreKind:
			var restoreList *velerov1.RestoreList
			if restoreList, listErr = veleroClient.Restores(VeleroNamespace).List(ctx, listOptions); listErr != nil {
				return listErr
			}
			if len(restoreList.Items) == 0 {
				// All selected items are already deleted
				return nil
			}
			// Delete all restores
			for _, restore := range restoreList.Items {
				deleteErr = veleroClient.Restores(VeleroNamespace).Delete(ctx, restore.GetName(), metav1.DeleteOptions{})
				if deleteErr != nil && !apierrors.IsNotFound(deleteErr) {
					return deleteErr
				}
			}
		}
	}
	return nil
}

// if backup.Spec.Schedule is set, then it is ScheduleBackup. otherwise, it is regular/ont-time backup.
func isScheduleBackup(backup *backupapi.Backup) bool {
	return len(backup.Spec.Schedule) != 0
}

func generateVeleroInstanceLabel(createdByLabel, creatorName, fleetName string) map[string]string {
	return map[string]string{
		createdByLabel:  creatorName,
		FleetLabel:      fleetName,
		FleetPluginName: plugin.BackupPluginName,
	}
}

func generateVeleroResourceObjectMeta(veleroResourceName string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      veleroResourceName,
		Namespace: VeleroNamespace,
		Labels:    labels,
	}
}

func generateVeleroResourceName(clusterName, creatorKind, creatorName string) string {
	return clusterName + "-" + creatorKind + "-" + creatorName
}

func migrateBackupCompleted(migrate *backupapi.Migrate) bool {
	return migrate.Status.Phase == backupapi.MigratePhaseSourceReady || migrate.Status.Phase == backupapi.MigratePhaseInProgress
}

// MostRecentCompletedBackup returns the most recent backup that's completed from a list of backups.
// origin from https://github.com/vmware-tanzu/velero/blob/release-1.12/pkg/controller/restore_controller.go
func MostRecentCompletedBackup(backups []velerov1.Backup) velerov1.Backup {
	sort.Slice(backups, func(i, j int) bool {
		var iStartTime, jStartTime time.Time
		if backups[i].Status.StartTimestamp != nil {
			iStartTime = backups[i].Status.StartTimestamp.Time
		}
		if backups[j].Status.StartTimestamp != nil {
			jStartTime = backups[j].Status.StartTimestamp.Time
		}
		return iStartTime.After(jStartTime)
	})

	for _, backup := range backups {
		if backup.Status.Phase == velerov1.BackupPhaseCompleted {
			return backup
		}
	}

	return velerov1.Backup{}
}

// GetCronInterval return the cron interval of a cron expression。
func GetCronInterval(cronExpr string) (time.Duration, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	nextRun := schedule.Next(now)
	nextNextRun := schedule.Next(nextRun)

	// Adding a 30-second delay to avoid timing issues.
	// Without this delay, we risk checking just before a new backup starts,
	// seeing the previous backup's "completed" status, and missing the new one.
	const delay = 30 * time.Second
	interval := nextNextRun.Sub(nextRun) + delay

	return interval, nil
}
