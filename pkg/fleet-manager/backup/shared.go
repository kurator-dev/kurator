/*
Copyright 2022-2025 Kurator Authors.
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

package backup

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/robfig/cron/v3"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	backupapi "kurator.dev/kurator/pkg/apis/backups/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	fleetmanager "kurator.dev/kurator/pkg/fleet-manager"
	"kurator.dev/kurator/pkg/fleet-manager/plugin"
)

const (
	// BackupNameLabel is the label key used to identify a schedule by name.
	BackupNameLabel = "kurator.dev/backup-name"
	// RestoreNameLabel is the label key used to identify a restore by name.
	RestoreNameLabel = "kurator.dev/restore-name"
	// MigrateNameLabel is the label key used to identify a migrate by name.
	MigrateNameLabel = "kurator.dev/migrate-name"

	BackupKind  = "backup"
	RestoreKind = "restore"
	MigrateKind = "migrate"

	// VeleroNamespace defines the default namespace where all Velero resources are created. It's a constant namespace used by Velero.
	VeleroNamespace = "velero"

	BackupFinalizer  = "backup.kurator.dev"
	RestoreFinalizer = "restore.kurator.dev"
	MigrateFinalizer = "migrate.kurator.dev"

	// StatusSyncInterval specifies the interval for requeueing when synchronizing status. It determines how frequently the status should be checked and updated.
	StatusSyncInterval = 30 * time.Second
)

// fetchDestinationClusters retrieves the clusters from the specified destination and filters them based on the 'Clusters' defined in the destination. It returns a map of selected clusters along with any error encountered during the process.
func fetchDestinationClusters(ctx context.Context, kubeClient client.Client, namespace string, destination backupapi.Destination) (map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster, error) {
	// Fetch fleet instance
	fleet := &fleetapi.Fleet{}
	fleetKey := client.ObjectKey{
		Namespace: namespace,
		Name:      destination.Fleet,
	}
	if err := kubeClient.Get(ctx, fleetKey, fleet); err != nil {
		return nil, fmt.Errorf("failed to retrieve fleet instance '%s' in namespace '%s': %w", destination.Fleet, namespace, err)
	}

	fleetClusters, err := fleetmanager.BuildFleetClusters(ctx, kubeClient, fleet)
	if err != nil {
		return nil, fmt.Errorf("failed to build fleet clusters from fleet instance '%s': %w", fleet.Name, err)
	}

	// If no destination.Clusters defined, return all clusters
	if len(destination.Clusters) == 0 {
		return fleetClusters, nil
	}

	selectedFleetCluster := make(map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster)
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

// buildVeleroBackupInstance constructs a Velero Backup instance configured to perform a backup operation on the specified cluster.
func buildVeleroBackupInstance(backupSpec *backupapi.BackupSpec, labels map[string]string, veleroBackupName string) *velerov1.Backup {
	veleroBackup := &velerov1.Backup{
		ObjectMeta: generateVeleroResourceObjectMeta(veleroBackupName, labels),
		Spec:       buildVeleroBackupSpec(backupSpec.Policy),
	}
	return veleroBackup
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

func newSyncVeleroTaskFunc(ctx context.Context, clusterAccess *fleetmanager.FleetCluster, obj client.Object) func() error {
	return func() error {
		return syncVeleroObj(ctx, clusterAccess, obj)
	}
}

func syncVeleroObj(ctx context.Context, cluster *fleetmanager.FleetCluster, veleroObj client.Object) error {
	// Get the client
	clusterClient := cluster.GetRuntimeClient()

	// create or update veleroRestore
	_, syncErr := controllerutil.CreateOrUpdate(ctx, clusterClient, veleroObj, func() error {
		// the veleroObj already contains the desired state, and there's no need for any additional modifications in this mutateFn.
		return nil
	})

	return syncErr
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

// deleteResourcesInClusters deletes instances of a Kubernetes resource based on the specified label key and value.
// It iterates over all destination clusters and performs the deletion.
// Parameters:
// - ctx: context to carry out the API requests
// - labelKey: the key of the label to filter the resources
// - labelValue: the value of the label to filter the resources
// - destinationClusters: a map containing information about the destination clusters
// - objList: an empty instance of the resource list object to be filled with retrieved data. It should implement client.ObjectList.
// Returns:
// - error: if any error occurs during the deletion process
// deleteResourcesInClusters deletes instances of a Kubernetes resource based on the specified label key and value.
func deleteResourcesInClusters(ctx context.Context, namespace, labelKey string, labelValue string, destinationClusters map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster, objList client.ObjectList) error {
	// Log setup
	log := ctrl.LoggerFrom(ctx)

	// Iterate over each destination cluster
	for clusterKey, clusterAccess := range destinationClusters {
		// List the resources using the helper function
		if err := listResourcesFromClusterClient(ctx, namespace, labelKey, labelValue, *clusterAccess, objList); err != nil {
			log.Error(err, "Failed to list resources in cluster", "ClusterName", clusterKey.Name, "Namespace", namespace, "LabelKey", labelKey, "LabelValue", labelValue)
			return err
		}

		// Extract Items using reflection
		itemsValue := reflect.ValueOf(objList).Elem().FieldByName("Items")
		if !itemsValue.IsValid() {
			err := fmt.Errorf("failed to extract 'Items' from object list using reflection")
			log.Error(err, "Reflection error")
			return err
		}

		for i := 0; i < itemsValue.Len(); i++ {
			item := itemsValue.Index(i).Addr().Interface().(client.Object)

			clusterClient := clusterAccess.GetRuntimeClient()

			if err := clusterClient.Delete(ctx, item); err != nil && !apierrors.IsNotFound(err) {
				log.Error(err, "Failed to delete resource in cluster", "ResourceName", item.GetName(), "ResourceNamespace", item.GetNamespace(), "ClusterName", clusterKey.Name)
				return err
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
		createdByLabel:               creatorName,
		fleetmanager.FleetLabel:      fleetName,
		fleetmanager.FleetPluginName: plugin.BackupPluginName,
	}
}

func generateVeleroResourceObjectMeta(veleroResourceName string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      veleroResourceName,
		Namespace: VeleroNamespace,
		Labels:    labels,
	}
}

// generateVeleroResourceName generate a name uniquely across object store
func generateVeleroResourceName(clusterName, creatorKind, creatorNamespace, creatorName string) string {
	return clusterName + "-" + creatorKind + "-" + creatorNamespace + "-" + creatorName
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

// getResourceFromClusterClient retrieves a specific Kubernetes resource from the provided cluster.
func getResourceFromClusterClient(ctx context.Context, name, namespace string, clusterAccess fleetmanager.FleetCluster, obj client.Object) error {
	clusterClient := clusterAccess.GetRuntimeClient()

	resourceKey := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	return clusterClient.Get(ctx, resourceKey, obj)
}

// listResourcesFromClusterClient retrieves resources from a cluster based on the provided namespace and label.
func listResourcesFromClusterClient(ctx context.Context, namespace string, labelKey string, labelValue string, clusterAccess fleetmanager.FleetCluster, objList client.ObjectList) error {
	// Create the cluster client
	clusterClient := clusterAccess.GetRuntimeClient()
	// Create the label selector
	labelSelector := labels.Set(map[string]string{labelKey: labelValue}).AsSelector()

	// List the resources
	opts := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labelSelector,
	}
	return clusterClient.List(ctx, objList, opts)
}

// fetchRestoreDestinationClusters retrieves the destination clusters for a restore operation.
// It first fetches the referred backup and then determines the destination clusters based on the restore and backup specifications:
// If the restore destination is not set, it returns the clusters from the backup destination.
// If set, it ensures that the restore destination is a subset of the backup destination.
//
// Returns:
// - string: The name of the fleet where the restore's set of fleetClusters resides.
// - map[ClusterKey]*FleetCluster: A map of cluster keys to fleet clusters.
// - error: An error object indicating any issues encountered during the operation.
func (r *RestoreManager) fetchRestoreDestinationClusters(ctx context.Context, restore *backupapi.Restore) (string, map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster, error) {
	// Retrieve the referred backup in the current Kurator host cluster
	key := client.ObjectKey{
		Name:      restore.Spec.BackupName,
		Namespace: restore.Namespace,
	}
	referredBackup := &backupapi.Backup{}
	if err := r.Client.Get(ctx, key, referredBackup); err != nil {
		return "", nil, fmt.Errorf("failed to retrieve the referred backup '%s': %w", restore.Spec.BackupName, err)
	}

	// Get the base clusters from the referred backup
	baseClusters, err := fetchDestinationClusters(ctx, r.Client, restore.Namespace, referredBackup.Spec.Destination)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch fleet clusters for the backup '%s': %w", referredBackup.Name, err)
	}

	// If the restore destination is not set, return the base fleet clusters directly
	if restore.Spec.Destination == nil {
		return referredBackup.Spec.Destination.Fleet, baseClusters, nil
	}

	// If the restore destination is set, try to get the clusters from the restore destination
	restoreClusters, err := fetchDestinationClusters(ctx, r.Client, restore.Namespace, *restore.Spec.Destination)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch restore clusters for the restore '%s': %w", restore.Name, err)
	}

	// Check the fleet and clusters between the restore and the referred backup
	if referredBackup.Spec.Destination.Fleet != restore.Spec.Destination.Fleet {
		// if we make sure only one fleet in one ns, this error will never happen
		return "", nil, errors.New("the restore destination fleet must be the same as the backup's")
	}

	// In our design, the restore destination must be a subset of the backup destination.
	if !isFleetClusterSubset(baseClusters, restoreClusters) {
		return "", nil, errors.New("the restore clusters must be a subset of the base clusters")
	}

	return restore.Spec.Destination.Fleet, restoreClusters, nil
}

// isFleetClusterSubset is the helper function to check if one set of clusters is a subset of another
func isFleetClusterSubset(baseClusters, subsetClusters map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster) bool {
	for key := range subsetClusters {
		if _, exists := baseClusters[key]; !exists {
			return false
		}
	}
	return true
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
		// in velero, the restore does not contain namespace scope filter
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

// allRestoreCompleted checks if all restore operations are completed by inspecting the phase of each RestoreDetails instance in the provided slice.
func allRestoreCompleted(clusterDetails []*backupapi.RestoreDetails) bool {
	for _, detail := range clusterDetails {
		if detail.RestoreStatusInCluster == nil || detail.RestoreStatusInCluster.Phase != velerov1.RestorePhaseCompleted {
			return false
		}
	}
	return true
}

// syncVeleroRestoreStatus synchronizes the status of Velero restore resources across different clusters.
// Note: Returns the modified ClusterDetails to capture internal changes due to Go's slice behavior.
func syncVeleroRestoreStatus(ctx context.Context, destinationClusters map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster, clusterDetails []*backupapi.RestoreDetails, creatorKind, creatorNamespace, creatorName string) ([]*backupapi.RestoreDetails, error) {
	log := ctrl.LoggerFrom(ctx)

	if clusterDetails == nil {
		clusterDetails = []*backupapi.RestoreDetails{}
	}

	// Initialize a map to store the velero restore status of each cluster currently recorded. The combination of detail.ClusterName, detail.ClusterKind, and detail.BackupNameInCluster uniquely identifies a Velero restore object.
	statusMap := make(map[string]*backupapi.RestoreDetails)
	for _, detail := range clusterDetails {
		key := fmt.Sprintf("%s-%s-%s", detail.ClusterName, detail.ClusterKind, detail.RestoreNameInCluster)
		statusMap[key] = detail
	}
	// Loop through each target cluster to retrieve the status of Velero restore resources using the client associated with the respective target cluster.
	for clusterKey, clusterAccess := range destinationClusters {
		name := generateVeleroResourceName(clusterKey.Name, creatorKind, creatorNamespace, creatorName)
		veleroRestore := &velerov1.Restore{}
		err := getResourceFromClusterClient(ctx, name, VeleroNamespace, *clusterAccess, veleroRestore)
		if err != nil {
			log.Error(err, "failed to get velero restore instance for sync status", "restoreName", name)
			return nil, err
		}

		key := fmt.Sprintf("%s-%s-%s", clusterKey.Name, clusterKey.Kind, veleroRestore.Name)
		if detail, exists := statusMap[key]; exists {
			detail.RestoreStatusInCluster = &veleroRestore.Status
		} else {
			currentRestoreDetails := &backupapi.RestoreDetails{
				ClusterName:            clusterKey.Name,
				ClusterKind:            clusterKey.Kind,
				RestoreNameInCluster:   veleroRestore.Name,
				RestoreStatusInCluster: &veleroRestore.Status,
			}
			clusterDetails = append(clusterDetails, currentRestoreDetails)
		}
	}

	return clusterDetails, nil
}

// isMigrateSourceReady checks if the 'SourceReadyCondition' of a Migrate object is set to 'True'.
func isMigrateSourceReady(migrate *backupapi.Migrate) bool {
	for _, condition := range migrate.Status.Conditions {
		if condition.Type == backupapi.SourceReadyCondition && condition.Status == "True" {
			return true
		}
	}
	return false
}

// buildVeleroBackupFromMigrate constructs a Velero Backup instance based on the provided migrate specification.
func buildVeleroBackupFromMigrate(migrateSpec *backupapi.MigrateSpec, labels map[string]string, veleroBackupName string) *velerov1.Backup {
	// Only consider migrateSpec.Policy.ResourceFilter and migrateSpec.Policy.OrderedResources when building the Velero backup.
	backupParam := &backupapi.BackupSpec{}
	if migrateSpec.Policy != nil {
		backupParam.Policy = &backupapi.BackupPolicy{
			ResourceFilter:   migrateSpec.Policy.ResourceFilter,
			OrderedResources: migrateSpec.Policy.OrderedResources,
		}
	}
	return buildVeleroBackupInstance(backupParam, labels, veleroBackupName)
}

// buildVeleroRestoreFromMigrate constructs a Velero Restore instance based on the provided migrate specification.
func buildVeleroRestoreFromMigrate(migrateSpec *backupapi.MigrateSpec, labels map[string]string, veleroBackupName, veleroRestoreName string) *velerov1.Restore {
	// Only consider migrateSpec.Policy.NamespaceMapping, migrateSpec.Policy.PreserveNodePorts, and migrateSpec.Policy.MigrateStatus when building the Velero restore.
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
