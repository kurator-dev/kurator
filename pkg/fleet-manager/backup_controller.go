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

	"github.com/pkg/errors"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	backupapi "kurator.dev/kurator/pkg/apis/backups/v1alpha1"
)

// BackupManager reconciles a Backup object
type BackupManager struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (b *BackupManager) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupapi.Backup{}).
		WithOptions(options).
		Complete(b)
}

func (b *BackupManager) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)
	backup := &backupapi.Backup{}

	if err := b.Client.Get(ctx, req.NamespacedName, backup); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("backup object not found", "backup", req)
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Initialize patch helper
	patchHelper, err := patch.NewHelper(backup, b.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to init patch helper for backup %s", req.NamespacedName)
	}
	// Setup deferred function to handle patching the object at the end of the reconciler
	defer func() {
		patchOpts := []patch.Option{}
		if err := patchHelper.Patch(ctx, backup, patchOpts...); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch %s  %s", backup.Name, req.NamespacedName)})
		}
	}()

	// Check and add finalizer if not present
	if !controllerutil.ContainsFinalizer(backup, BackupFinalizer) {
		controllerutil.AddFinalizer(backup, BackupFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deletion
	if backup.GetDeletionTimestamp() != nil {
		return b.reconcileDeleteBackup(ctx, backup)
	}

	// Handle the main reconcile logic
	return b.reconcileBackup(ctx, backup)
}

// reconcileBackup handles the main reconcile logic for a Backup object.
func (b *BackupManager) reconcileBackup(ctx context.Context, backup *backupapi.Backup) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch destination clusters
	destinationClusters, err := fetchDestinationClusters(ctx, b.Client, backup.Namespace, backup.Spec.Destination)
	if err != nil {
		log.Error(err, "failed to fetch destination clusters for backup", "backupName", backup.Name)
		return ctrl.Result{}, err
	}
	// Apply velero backup resource in target clusters
	result, err := b.reconcileBackupResources(ctx, backup, destinationClusters)
	if err != nil || result.Requeue || result.RequeueAfter > 0 {
		return result, err
	}
	// Collect velero backup resource status to current backup
	return b.reconcileBackupStatus(ctx, backup, destinationClusters)
}

// reconcileBackupResources converts the backup resources into velero backup resources that can be used by Velero on the target clusters, and applies each of these backup resources to the respective target clusters.
func (b *BackupManager) reconcileBackupResources(ctx context.Context, backup *backupapi.Backup, destinationClusters map[ClusterKey]*fleetCluster) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	backupLabel := generateVeleroInstanceLabel(BackupNameLabel, backup.Name, backup.Spec.Destination.Fleet)

	if isScheduleBackup(backup) {
		// Handle scheduled backups
		for clusterKey, clusterAccess := range destinationClusters {
			veleroScheduleName := generateVeleroResourceName(clusterKey.Name, BackupKind, backup.Name)
			veleroSchedule := buildVeleroScheduleInstance(&backup.Spec, backupLabel, veleroScheduleName)
			if err := syncVeleroObj(ctx, clusterKey, clusterAccess, veleroSchedule); err != nil {
				log.Error(err, "failed to create velero schedule instance for backup", "backupName", backup.Name)
				return ctrl.Result{}, err
			}
		}
	} else {
		// Handle one time backups
		for clusterKey, clusterAccess := range destinationClusters {
			veleroBackupName := generateVeleroResourceName(clusterKey.Name, BackupKind, backup.Name)
			veleroBackup := buildVeleroBackupInstance(&backup.Spec, backupLabel, veleroBackupName)
			if err := syncVeleroObj(ctx, clusterKey, clusterAccess, veleroBackup); err != nil {
				log.Error(err, "failed to create velero backup instance for backup", "backupName", backup.Name)
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

// reconcileBackupStatus updates the synchronization status of each backup resource.
func (b *BackupManager) reconcileBackupStatus(ctx context.Context, backup *backupapi.Backup, destinationClusters map[ClusterKey]*fleetCluster) (ctrl.Result, error) {
	// Initialize a map to store the status of each cluster currently recorded. The combination of detail.ClusterName, detail.ClusterKind, and detail.BackupNameInCluster uniquely identifies a Velero backup object.
	statusMap := make(map[string]*backupapi.BackupDetails)
	for _, detail := range backup.Status.Details {
		key := fmt.Sprintf("%s-%s-%s", detail.ClusterName, detail.ClusterKind, detail.BackupNameInCluster)
		statusMap[key] = detail
	}
	if isScheduleBackup(backup) {
		return b.reconcileScheduleBackupStatus(ctx, backup, destinationClusters, statusMap)
	} else {
		return b.reconcileOneTimeBackupStatus(ctx, backup, destinationClusters, statusMap)
	}
}

// reconcileOneTimeBackupStatus updates the status of a one-time Backup object by checking the status of corresponding Velero backup resources in each target cluster.
// It determines whether to requeue the reconciliation based on the completion status of all Velero backup resources.
func (b *BackupManager) reconcileOneTimeBackupStatus(ctx context.Context, backup *backupapi.Backup, destinationClusters map[ClusterKey]*fleetCluster, statusMap map[string]*backupapi.BackupDetails) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Loop through each target cluster to retrieve the status of Velero backup resources using the client associated with the respective target cluster.
	for clusterKey, clusterAccess := range destinationClusters {
		name := generateVeleroResourceName(clusterKey.Name, BackupKind, backup.Name)
		veleroBackup := &velerov1.Backup{}

		// Use the client of the target cluster to get the status of Velero backup resources
		err := getResourceFromClusterClient(ctx, name, VeleroNamespace, *clusterAccess, veleroBackup)
		if err != nil {
			log.Error(err, "failed to create velero backup instance for sync one time backup status", "backupName", backup.Name)
			return ctrl.Result{}, err
		}

		key := fmt.Sprintf("%s-%s-%s", clusterKey.Name, clusterKey.Kind, veleroBackup.Name)
		if detail, exists := statusMap[key]; exists {
			// If a matching entry is found, update the existing BackupDetails object with the new status.
			detail.BackupStatusInCluster = &veleroBackup.Status
		} else {
			// If no matching entry is found, create a new BackupDetails object and append it to the backup's status details.
			currentBackupDetails := &backupapi.BackupDetails{
				ClusterName:           clusterKey.Name,
				ClusterKind:           clusterKey.Kind,
				BackupNameInCluster:   veleroBackup.Name,
				BackupStatusInCluster: &veleroBackup.Status,
			}
			backup.Status.Details = append(backup.Status.Details, currentBackupDetails)
		}
	}

	// Determine whether to requeue the reconciliation based on the completion status of all Velero backup resources.
	// If all backups are complete, exit directly without requeuing.
	// Otherwise, requeue the reconciliation after a specified interval (StatusSyncInterval).
	if allBackupsCompleted(backup.Status) {
		return ctrl.Result{}, nil
	} else {
		return ctrl.Result{RequeueAfter: StatusSyncInterval}, nil
	}
}

// reconcileScheduleBackupStatus manages the status synchronization for scheduled Backup objects.
// If the backup type is "schedule", new backups will be continuously generated, hence the status synchronization will be executed continuously.
func (b *BackupManager) reconcileScheduleBackupStatus(ctx context.Context, schedule *backupapi.Backup, destinationClusters map[ClusterKey]*fleetCluster, statusMap map[string]*backupapi.BackupDetails) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Loop through each target cluster to retrieve the status of Velero backup created by schedule resources using the client associated with the respective target cluster.
	for clusterKey, clusterAccess := range destinationClusters {
		name := generateVeleroResourceName(clusterKey.Name, BackupKind, schedule.Name)
		veleroSchedule := &velerov1.Schedule{}
		// Use the client of the target cluster to get the status of Velero backup resources
		err := getResourceFromClusterClient(ctx, name, VeleroNamespace, *clusterAccess, veleroSchedule)
		if err != nil {
			log.Error(err, "Unable to get velero schedule", "scheduleName", name)
			return ctrl.Result{}, err
		}

		// Fetch all velero backups created by velero schedule
		backupList := &velerov1.BackupList{}
		listErr := listResourcesFromClusterClient(ctx, VeleroNamespace, velerov1.ScheduleNameLabel, veleroSchedule.Name, *clusterAccess, backupList)
		if listErr != nil {
			log.Info("Unable to list velero backups for velero schedule", "scheduleName", veleroSchedule.Name)
			return ctrl.Result{}, listErr
		}

		// Fetch most recent completed backup
		veleroBackup := MostRecentCompletedBackup(backupList.Items)
		if len(veleroBackup.Name) == 0 {
			// If a schedule backup record cannot be found, the potential reasons are:
			// 1. The backup task hasn't been triggered by schedule.
			// 2. An issue occurred, but we can not get information directly from the status of schedules.velero.io
			log.Info("No completed backups found for schedule", "scheduleName", veleroSchedule.Name)
		}

		// Sync schedule backup status with most recent complete backup
		key := fmt.Sprintf("%s-%s-%s", clusterKey.Name, clusterKey.Kind, veleroBackup.Name)
		if detail, exists := statusMap[key]; exists {
			// If a matching entry is found, update the existing BackupDetails object with the new status.
			detail.BackupStatusInCluster = &veleroBackup.Status
		} else {
			// If no matching entry is found, create a new BackupDetails object and append it to the schedule's status details.
			currentBackupDetails := &backupapi.BackupDetails{
				ClusterName:           clusterKey.Name,
				ClusterKind:           clusterKey.Kind,
				BackupNameInCluster:   veleroBackup.Name,
				BackupStatusInCluster: &veleroBackup.Status,
			}
			schedule.Status.Details = append(schedule.Status.Details, currentBackupDetails)
		}
	}

	if allBackupsCompleted(schedule.Status) {
		// Get the next reconcile interval
		cronInterval, err := GetCronInterval(schedule.Spec.Schedule)
		if err != nil {
			log.Error(err, "failed to get cron Interval of backup.spec.schedule", "backupName", schedule.Name, "cronExpression", schedule.Spec.Schedule)
			return ctrl.Result{}, err
		}
		// If all backups are complete,requeue the reconciliation after a long cronInterval.
		return ctrl.Result{RequeueAfter: cronInterval}, nil
	}
	// If not all backups are complete, requeue the reconciliation after a short StatusSyncInterval.
	return ctrl.Result{RequeueAfter: StatusSyncInterval}, nil
}

// reconcileDeleteBackup handles the deletion process of a Backup object.
func (b *BackupManager) reconcileDeleteBackup(ctx context.Context, backup *backupapi.Backup) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch backup destination clusters
	destinationClusters, err := fetchDestinationClusters(ctx, b.Client, backup.Namespace, backup.Spec.Destination)
	if err != nil {
		log.Error(err, "failed to fetch destination clusters when delete backup", "backupName", backup.Name)
		controllerutil.RemoveFinalizer(backup, BackupFinalizer)
		log.Info("Removed finalizer due to fetch destination clusters error", "backupName", backup.Name)
		return ctrl.Result{}, err
	}

	var objList client.ObjectList
	if isScheduleBackup(backup) {
		objList = &velerov1.ScheduleList{}
	} else {
		objList = &velerov1.BackupList{}
	}

	// Delete all related velero schedule or backup instance
	if err := deleteResourcesInClusters(ctx, VeleroNamespace, BackupNameLabel, backup.Name, destinationClusters, objList); err != nil {
		log.Error(err, "failed to delete velero schedule or backup Instances when delete backup", "backupName", backup.Name)
		return ctrl.Result{}, err
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(backup, BackupFinalizer)

	return ctrl.Result{}, nil
}
