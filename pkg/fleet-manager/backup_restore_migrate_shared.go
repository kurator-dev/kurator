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
	"fmt"
	"reflect"
	"sort"
	"sync"
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
func fetchDestinationClusters(ctx context.Context, kubeClient client.Client, namespace string, destination backupapi.Destination) (map[ClusterKey]*fleetCluster, error) {
	// Fetch fleet instance
	fleet := &fleetapi.Fleet{}
	fleetKey := client.ObjectKey{
		Namespace: namespace,
		Name:      destination.Fleet,
	}
	if err := kubeClient.Get(ctx, fleetKey, fleet); err != nil {
		return nil, fmt.Errorf("failed to retrieve fleet instance '%s' in namespace '%s': %w", destination.Fleet, namespace, err)
	}

	fleetClusters, err := buildFleetClusters(ctx, kubeClient, fleet)
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

func newSyncVeleroTaskFunc(ctx context.Context, clusterAccess *fleetCluster, obj client.Object) func() error {
	return func() error {
		return syncVeleroObj(ctx, clusterAccess, obj)
	}
}

func syncVeleroObj(ctx context.Context, cluster *fleetCluster, veleroObj client.Object) error {
	// Get the client
	clusterClient := cluster.client.CtrlRuntimeClient()

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
func deleteResourcesInClusters(ctx context.Context, namespace, labelKey string, labelValue string, destinationClusters map[ClusterKey]*fleetCluster, objList client.ObjectList) error {
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

			clusterClient := clusterAccess.client.CtrlRuntimeClient()

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
func getResourceFromClusterClient(ctx context.Context, name, namespace string, clusterAccess fleetCluster, obj client.Object) error {
	clusterClient := clusterAccess.client.CtrlRuntimeClient()

	resourceKey := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	return clusterClient.Get(ctx, resourceKey, obj)
}

// listResourcesFromClusterClient retrieves resources from a cluster based on the provided namespace and label.
func listResourcesFromClusterClient(ctx context.Context, namespace string, labelKey string, labelValue string, clusterAccess fleetCluster, objList client.ObjectList) error {
	// Create the cluster client
	clusterClient := clusterAccess.client.CtrlRuntimeClient()
	// Create the label selector
	labelSelector := labels.Set(map[string]string{labelKey: labelValue}).AsSelector()

	// List the resources
	opts := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labelSelector,
	}
	return clusterClient.List(ctx, objList, opts)
}

// parallelProcess runs the provided tasks concurrently and collects any errors.
func parallelProcess(tasks []func() error) []error {
	var errs []error
	var errMutex sync.Mutex
	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(task func() error) {
			defer wg.Done()
			if err := task(); err != nil {
				errMutex.Lock()
				errs = append(errs, err)
				errMutex.Unlock()
			}
		}(task)
	}

	wg.Wait()
	return errs
}
