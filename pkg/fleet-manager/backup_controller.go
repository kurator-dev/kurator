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
	"time"

	"github.com/pkg/errors"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	backupapi "kurator.dev/kurator/pkg/apis/backups/v1alpha1"
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

	VeleroBackupKind   = "velero-backup"
	VeleroScheduleKind = "velero-schedule"
	VeleroRestoreKind  = "velero-restore"

	// VeleroNamespace defines the default namespace where all Velero resources are created. It's a constant namespace used by Velero.
	VeleroNamespace = "velero"

	BackupFinalizer = "backup.kurator.dev"

	// StatusSyncInterval specifies the interval for requeueing when synchronizing status. It determines how frequently the status should be checked and updated.
	StatusSyncInterval = 30 * time.Second
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
		Watches(&source.Kind{Type: &backupapi.Restore{}}, &handler.EnqueueRequestForObject{}).
		Watches(&source.Kind{Type: &backupapi.Migrate{}}, &handler.EnqueueRequestForObject{}).
		WithOptions(options).
		Complete(b)
}

func (b *BackupManager) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	var currentObject client.Object
	var crdType string
	counter := 0

	log := ctrl.LoggerFrom(ctx)

	// Define the list of CRDs that this reconciler is responsible for
	crds := []struct {
		obj     client.Object
		typeStr string
	}{
		{&backupapi.Backup{}, BackupKind},
		{&backupapi.Restore{}, RestoreKind},
		{&backupapi.Migrate{}, MigrateKind},
	}

	// Loop through each CRD type and try to get the object with the given namespace and name
	for _, crd := range crds {
		err := b.Get(ctx, req.NamespacedName, crd.obj)
		if err == nil {
			counter++
			currentObject = crd.obj
			crdType = crd.typeStr
		} else if !apierrors.IsNotFound(err) {
			log.Error(err, fmt.Sprintf("failed to get %s", crd.typeStr), "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, err
		}
	}

	// Check if multiple instances of CRDs (backup, restore, migrate) with the same namespace and name were found.
	// This scenario is prohibited in our current design as it could lead to ambiguous behavior during the reconciliation process.
	if counter > 1 {
		log.Error(nil, "multiple CRDs(backup, restore, migrate) with the same namespace and name were found", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, nil
	}

	// Check if no instances of the CRDs (backup, restore, migrate) were found with the specified namespace and name.
	if counter == 0 {
		log.Info("no backup, restore or migrate found for the given namespace and name", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, nil
	}

	// Initialize patch helper
	patchHelper, err := patch.NewHelper(currentObject, b.Client)
	if err != nil {
		log.Error(err, "failed to init patch helper")
	}
	// Setup deferred function to handle patching the object at the end of the reconciler
	defer func() {
		patchOpts := []patch.Option{}
		if err := patchHelper.Patch(ctx, currentObject, patchOpts...); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch %s  %s", crdType, req.NamespacedName)})
		}
	}()

	// Check and add finalizer if not present
	if !controllerutil.ContainsFinalizer(currentObject, BackupFinalizer) {
		controllerutil.AddFinalizer(currentObject, BackupFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deletion
	if currentObject.GetDeletionTimestamp() != nil {
		switch crdType {
		case "backup":
			return b.reconcileDeleteBackup(ctx, currentObject.(*backupapi.Backup))
		case "restore":
			return b.reconcileDeleteRestore(ctx, currentObject.(*backupapi.Restore))
		case "migrate":
			return b.reconcileDeleteMigrate(ctx, currentObject.(*backupapi.Migrate))
		}
	}

	// Handle the main reconcile logic based on the type of the CRD object found
	switch crdType {
	case "backup":
		return b.reconcileBackup(ctx, currentObject.(*backupapi.Backup))
	case "restore":
		return b.reconcileRestore(ctx, currentObject.(*backupapi.Restore))
	case "migrate":
		return b.reconcileMigrate(ctx, currentObject.(*backupapi.Migrate))
	}

	log.Error(errors.New("unreachable code reached"), "This should not happen")
	return ctrl.Result{}, nil
}

// reconcileBackup handles the main reconcile logic for a Backup object.
func (b *BackupManager) reconcileBackup(ctx context.Context, backup *backupapi.Backup) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch destination clusters
	destinationClusters, err := b.fetchDestinationClusters(ctx, backup.Namespace, backup.Spec.Destination)
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
		for clusterKey, fleetCluster := range destinationClusters {
			veleroScheduleName := generateVeleroResourceName(clusterKey.Name, BackupKind, backup.Name)
			veleroSchedule := buildVeleroScheduleInstance(&backup.Spec, backupLabel, veleroScheduleName)
			if err := createVeleroScheduleInstance(fleetCluster, veleroSchedule); err != nil {
				log.Error(err, "failed to create velero schedule instance for backup", "backupName", backup.Name)
				return ctrl.Result{}, err
			}
		}
	} else {
		// Handle one time backups
		for clusterKey, fleetCluster := range destinationClusters {
			veleroBackupName := generateVeleroResourceName(clusterKey.Name, BackupKind, backup.Name)
			veleroBackup := buildVeleroBackupInstance(&backup.Spec, backupLabel, veleroBackupName)
			if err := createVeleroBackupInstance(fleetCluster, veleroBackup); err != nil {
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
	for clusterKey, fleetCluster := range destinationClusters {
		name := generateVeleroResourceName(clusterKey.Name, BackupKind, backup.Name)
		// Use the client of the target cluster to get the status of Velero backup resources
		veleroBackup, err := fleetCluster.client.VeleroClient().VeleroV1().Backups(VeleroNamespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log.Error(err, "failed to create velero backup instance for sync one time backup status", "backupName", backup.Name)
			return ctrl.Result{}, err
		}

		log.Info("check veleroBackup status", "velero-backup-name", name, "velero-backup-status", veleroBackup.Status)

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
		log.Info("all backup status is completed!", "backup", backup.Name)
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
	for clusterKey, fleetCluster := range destinationClusters {
		name := generateVeleroResourceName(clusterKey.Name, BackupKind, schedule.Name)

		// Use the client of the target cluster to get the status of Velero backup resources
		veleroSchedule, err := fleetCluster.client.VeleroClient().VeleroV1().Schedules(VeleroNamespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log.Error(err, "Unable to get velero schedule", "scheduleName", name)
			return ctrl.Result{}, err
		}

		// Fetch all velero backups created by velero schedule
		labelSelector := metav1.LabelSelector{
			MatchLabels: map[string]string{
				velerov1.ScheduleNameLabel: veleroSchedule.Name,
			},
		}
		listOptions := metav1.ListOptions{
			LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		}
		backupList, listErr := fleetCluster.client.VeleroClient().VeleroV1().Backups(VeleroNamespace).List(ctx, listOptions)
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
		log.Info("all backup status is completed!", "backup", schedule.Name)
		return ctrl.Result{RequeueAfter: cronInterval}, nil
	}
	// If not all backups are complete, requeue the reconciliation after a short StatusSyncInterval.
	return ctrl.Result{RequeueAfter: StatusSyncInterval}, nil
}

// reconcileRestore handles the main reconcile logic for a Restore object.
func (b *BackupManager) reconcileRestore(ctx context.Context, restore *backupapi.Restore) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	fleetName, destinationClusters, err := b.fetchRestoreDestinationClusters(ctx, restore)
	if err != nil {
		log.Error(err, "failed to fetch destination clusters for restore", "restoreName", restore.Name)
		return ctrl.Result{}, err
	}
	// Apply restore resource in target clusters
	result, err := b.reconcileRestoreResources(ctx, restore, destinationClusters, fleetName)
	if err != nil || result.Requeue || result.RequeueAfter > 0 {
		return result, err
	}
	// Collect target clusters backup resource status to current restore
	_, result, err = b.reconcileVeleroRestoreStatus(ctx, destinationClusters, restore.Status.Details, RestoreKind, restore.Name)
	return result, err
}

// reconcileBackupResources converts the backup resources into VeleroBackup resources that can be used by Velero on the target clusters, and applies each of these resources to the respective target clusters.
func (b *BackupManager) reconcileRestoreResources(ctx context.Context, restore *backupapi.Restore, destinationClusters map[ClusterKey]*fleetCluster, fleetName string) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	restoreLabels := generateVeleroInstanceLabel(RestoreNameLabel, restore.Name, fleetName)
	for clusterKey, fleetCluster := range destinationClusters {
		veleroBackupName := generateVeleroResourceName(clusterKey.Name, BackupKind, restore.Spec.BackupName)
		veleroRestoreName := generateVeleroResourceName(clusterKey.Name, RestoreKind, restore.Name)
		veleroRestore := buildVeleroRestoreInstance(&restore.Spec, restoreLabels, veleroBackupName, veleroRestoreName)
		if err := createVeleroRestoreInstance(fleetCluster, veleroRestore); err != nil {
			log.Error(err, "failed to create velero restore instance for restore", "restoreName", restore.Name, "veleroRestoreName", veleroRestore)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// reconcileVeleroRestoreStatus synchronizes the status of Velero restore resources across different clusters.
// the returns value boolean indicating whether all restores are completed (true) or not (false).
func (b *BackupManager) reconcileVeleroRestoreStatus(ctx context.Context, destinationClusters map[ClusterKey]*fleetCluster, ClusterDetails []*backupapi.RestoreDetails, restoreCreatorKind, restoreCreatorName string) (bool, ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Initialize a map to store the velero restore status of each cluster currently recorded. The combination of detail.ClusterName, detail.ClusterKind, and detail.BackupNameInCluster uniquely identifies a Velero restore object.
	statusMap := make(map[string]*backupapi.RestoreDetails)
	for _, detail := range ClusterDetails {
		key := fmt.Sprintf("%s-%s-%s", detail.ClusterName, detail.ClusterKind, detail.RestoreNameInCluster)
		statusMap[key] = detail
	}
	// Loop through each target cluster to retrieve the status of Velero restore resources using the client associated with the respective target cluster.
	for clusterKey, fleetCluster := range destinationClusters {
		name := generateVeleroResourceName(clusterKey.Name, restoreCreatorKind, restoreCreatorName)
		veleroRestore, err := fleetCluster.client.VeleroClient().VeleroV1().Restores(VeleroNamespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log.Error(err, "failed to get velero restore instance for sync status", "restoreName", name)
			return false, ctrl.Result{}, err
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
			ClusterDetails = append(ClusterDetails, currentRestoreDetails)
		}
	}

	// Determine whether to requeue the reconciliation based on the completion status of all Velero restore resources.
	// If all restore are complete, exit directly without requeuing.
	// Otherwise, requeue the reconciliation after StatusSyncInterval.
	if allRestoreCompleted(ClusterDetails) {
		return true, ctrl.Result{}, nil
	} else {
		return false, ctrl.Result{RequeueAfter: StatusSyncInterval}, nil
	}
}

// reconcileMigrate handles the main reconcile logic for a Migrate object.
func (b *BackupManager) reconcileMigrate(ctx context.Context, migrate *backupapi.Migrate) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Update the status
	phase := migrate.Status.Phase
	if len(phase) == 0 || phase == backupapi.MigratePhaseNew {
		migrate.Status.Phase = backupapi.MigratePhaseWaitingForSource
		log.Info("Migrate Phase changes", "phase", backupapi.MigratePhaseWaitingForSource)
	}

	// The actual migration operation can be divided into two stages: the backup stage and the restore stage.
	// The current stage is determined by checking the status of the migration.
	// If the backup for the migration is not completed or hasn't started, the reconcileMigrateBackup function is executed first.
	if !migrateBackupCompleted(migrate) {
		result, err := b.reconcileMigrateBackup(ctx, migrate)
		if err != nil || result.Requeue || result.RequeueAfter > 0 {
			return result, err
		}
	}
	return b.reconcileMigrateRestore(ctx, migrate)
}

// reconcileMigrateBackup orchestrates the backup process during migration.
func (b *BackupManager) reconcileMigrateBackup(ctx context.Context, migrate *backupapi.Migrate) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch details of the source cluster for migration
	sourceClusterName, sourceClusterNameKind, sourceClusterAccess, err := b.fetchMigrateSourceClusters(ctx, migrate.Namespace, migrate.Spec.SourceCluster)
	if err != nil {
		log.Error(err, "Failed to fetch source cluster details for migration")
		return ctrl.Result{}, fmt.Errorf("fetching source cluster details: %w", err)
	}

	// Construct labels and backup resource details for migration
	migrateLabel := generateVeleroInstanceLabel(MigrateNameLabel, migrate.Name, migrate.Spec.SourceCluster.Fleet)
	sourceBackupName := generateVeleroResourceName(sourceClusterName, MigrateKind, migrate.Name)
	sourceBackup := buildVeleroBackupInstanceUsingMigrate(&migrate.Spec, migrateLabel, sourceBackupName)

	// Attempt to create the backup resource
	if err := createVeleroBackupInstance(sourceClusterAccess, sourceBackup); err != nil {
		log.Error(err, "Failed to create backup resource for migration", "backupName", sourceBackupName)
		return ctrl.Result{}, fmt.Errorf("creating backup resource: %w", err)
	}

	// Retrieve the status of the created backup resource
	veleroBackup, err := sourceClusterAccess.client.VeleroClient().VeleroV1().Backups(VeleroNamespace).Get(ctx, sourceBackupName, metav1.GetOptions{})
	if err != nil {
		log.Error(err, "Failed to retrieve status of backup resource", "backupName", sourceBackupName)
		return ctrl.Result{}, fmt.Errorf("retrieving backup status: %w", err)
	}
	log.Info("Migration phase updated", "phase", backupapi.MigratePhaseSourceReady)

	// Update migration status based on the backup details
	currentBackupDetails := &backupapi.BackupDetails{
		ClusterName:           sourceClusterName,
		ClusterKind:           sourceClusterNameKind,
		BackupNameInCluster:   veleroBackup.Name,
		BackupStatusInCluster: &veleroBackup.Status,
	}
	migrate.Status.SourceClusterStatus = currentBackupDetails

	if veleroBackup.Status.Phase == velerov1.BackupPhaseCompleted {
		migrate.Status.Phase = backupapi.MigratePhaseSourceReady
		log.Info("Migration phase updated", "phase", backupapi.MigratePhaseSourceReady)
	}

	return ctrl.Result{}, nil
}

// reconcileMigrateRestore handles the restore stage of the migration process.
func (b *BackupManager) reconcileMigrateRestore(ctx context.Context, migrate *backupapi.Migrate) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	targetClusters, err := b.fetchDestinationClusters(ctx, migrate.Namespace, migrate.Spec.TargetClusters)
	if err != nil {
		log.Error(err, "Failed to fetch target clusters for migration")
		return ctrl.Result{}, fmt.Errorf("fetching target clusters: %w", err)
	}

	if migrate.Status.Phase == backupapi.MigratePhaseSourceReady {
		migrate.Status.Phase = backupapi.MigratePhaseInProgress
		log.Info("Migrate Phase changes", "phase", backupapi.MigratePhaseInProgress)
		// ensure that the restore point is created after the backup.
		time.Sleep(1 * time.Second)
	}

	// referredBackupName is same in different target clusters velero restore. SourceCluster only has one cluster, so the cluster[0].name is the real name of SourceCluster
	referredBackupName := generateVeleroResourceName(migrate.Spec.SourceCluster.Clusters[0].Name, MigrateKind, migrate.Name)
	// Handle create target clusters velero restore
	restoreLabel := generateVeleroInstanceLabel(MigrateNameLabel, migrate.Name, migrate.Spec.TargetClusters.Fleet)
	for clusterKey, clusterAccess := range targetClusters {
		veleroRestoreName := generateVeleroResourceName(clusterKey.Name, MigrateKind, migrate.Name)
		veleroRestore := buildVeleroRestoreInstanceUsingMigrate(&migrate.Spec, restoreLabel, referredBackupName, veleroRestoreName)
		if err := createVeleroRestoreInstance(clusterAccess, veleroRestore); err != nil {
			log.Error(err, "Failed to creating Velero restore instance", "restoreName", veleroRestoreName)
			return ctrl.Result{}, fmt.Errorf("creating Velero restore instance for cluster %s: %w", clusterKey.Name, err)
		}
	}

	allCompleted, result, err := b.reconcileVeleroRestoreStatus(ctx, targetClusters, migrate.Status.TargetClustersStatus, MigrateKind, migrate.Name)
	if err != nil {
		log.Error(err, "Failed to reconcile Velero restore status")
		return result, fmt.Errorf("reconciling Velero restore status: %w", err)
	}

	if allCompleted {
		migrate.Status.Phase = backupapi.MigratePhaseCompleted
		log.Info("Migrate Phase changes", "phase", backupapi.MigratePhaseCompleted)
	}

	return result, nil
}

// reconcileDeleteBackup handles the deletion process of a Backup object.
func (b *BackupManager) reconcileDeleteBackup(ctx context.Context, backup *backupapi.Backup) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch backup destination clusters
	destinationClusters, err := b.fetchDestinationClusters(ctx, backup.Namespace, backup.Spec.Destination)
	if err != nil {
		log.Error(err, "failed to fetch destination clusters when delete backup", "backupName", backup.Name)
		return ctrl.Result{}, err
	}

	if isScheduleBackup(backup) {
		// Delete all related velero schedule instance
		if err := deleteCRDInstances(ctx, BackupNameLabel, backup.Name, destinationClusters, VeleroScheduleKind); err != nil {
			log.Error(err, "failed to delete velero schedule Instances when delete backup", "backupName", backup.Name)
			return ctrl.Result{}, err
		}
	} else {
		// Delete all related velero backup instance
		if err := deleteCRDInstances(ctx, BackupNameLabel, backup.Name, destinationClusters, VeleroBackupKind); err != nil {
			log.Error(err, "failed to delete velero backup Instances when delete backup", "backupName", backup.Name)
			return ctrl.Result{}, err
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(backup, BackupFinalizer)

	return ctrl.Result{}, nil
}

// reconcileDeleteRestore handles the deletion process of a Migrate object.
func (b *BackupManager) reconcileDeleteRestore(ctx context.Context, restore *backupapi.Restore) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch backup destination clusters
	_, destinationClusters, err := b.fetchRestoreDestinationClusters(ctx, restore)
	if err != nil {
		log.Error(err, "failed to fetch destination clusters when delete restore", "restoreName", restore.Name)
		return ctrl.Result{}, err
	}

	// Delete all related velero backup instance
	if err := deleteCRDInstances(ctx, RestoreNameLabel, restore.Name, destinationClusters, VeleroRestoreKind); err != nil {
		log.Error(err, "failed to delete velero restore Instances when delete restore", "restoreName", restore.Name)
		return ctrl.Result{}, err
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(restore, BackupFinalizer)

	return ctrl.Result{}, nil
}

// reconcileDeleteMigrate handles the deletion process of a Migrate object.
func (b *BackupManager) reconcileDeleteMigrate(ctx context.Context, migrate *backupapi.Migrate) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch source cluster
	sourceCluster, err := b.fetchDestinationClusters(ctx, migrate.Namespace, migrate.Spec.SourceCluster)
	if err != nil {
		log.Error(err, "failed to fetch source clusters when delete migrate", "migrateName", migrate.Name)
		return ctrl.Result{}, err
	}

	// Delete related velero backup instance
	if err := deleteCRDInstances(ctx, MigrateNameLabel, migrate.Name, sourceCluster, VeleroBackupKind); err != nil {
		log.Error(err, "failed to delete velero migrate Instances when delete migrate", "restoreName", migrate.Name)
		return ctrl.Result{}, err
	}

	// Fetch target clusters
	targetClusters, err := b.fetchDestinationClusters(ctx, migrate.Namespace, migrate.Spec.SourceCluster)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to fetch target clusters for migrate: %w", err)
	}

	// Delete all related velero restore instance
	if err := deleteCRDInstances(ctx, MigrateNameLabel, migrate.Name, targetClusters, VeleroRestoreKind); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to delete related %s instances: %w", "migrate", err)
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(migrate, BackupFinalizer)

	return ctrl.Result{}, nil
}
