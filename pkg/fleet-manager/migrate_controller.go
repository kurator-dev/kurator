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
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	backupapi "kurator.dev/kurator/pkg/apis/backups/v1alpha1"
)

// MigrateManager reconciles a Migrate object
type MigrateManager struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (m *MigrateManager) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupapi.Migrate{}).
		WithOptions(options).
		Complete(m)
}

func (m *MigrateManager) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)
	migrate := &backupapi.Migrate{}

	if err := m.Client.Get(ctx, req.NamespacedName, migrate); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("migrate is not exist", "migrate", req)
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Initialize patch helper
	patchHelper, err := patch.NewHelper(migrate, m.Client)
	if err != nil {
		log.Error(err, "failed to init patch helper")
	}
	// Setup deferred function to handle patching the object at the end of the reconciler
	defer func() {
		patchOpts := []patch.Option{
			patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
				backupapi.SourceReadyCondition,
			}},
		}

		if err := patchHelper.Patch(ctx, migrate, patchOpts...); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch %s  %s", migrate.Name, req.NamespacedName)})
		}
	}()

	// Check and add finalizer if not present
	if !controllerutil.ContainsFinalizer(migrate, MigrateFinalizer) {
		controllerutil.AddFinalizer(migrate, MigrateFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deletion
	if migrate.GetDeletionTimestamp() != nil {
		return m.reconcileDeleteMigrate(ctx, migrate)
	}

	// Handle the main reconcile logic
	return m.reconcileMigrate(ctx, migrate)
}

// reconcileMigrate handles the main reconcile logic for a Migrate object.
func (m *MigrateManager) reconcileMigrate(ctx context.Context, migrate *backupapi.Migrate) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Update the migrate phase
	phase := migrate.Status.Phase
	if len(phase) == 0 || phase == backupapi.MigratePhasePending {
		migrate.Status.Phase = backupapi.MigratePhaseWaitingForSource
		log.Info("Migrate Phase changes", "phase", backupapi.MigratePhaseWaitingForSource)
	}

	// The actual migration operation can be divided into two stages
	// 1 the backup stage
	result, err := m.reconcileMigrateBackup(ctx, migrate)
	if err != nil || result.Requeue || result.RequeueAfter > 0 {
		return result, err
	}

	// 2 the restore stage.
	return m.reconcileMigrateRestore(ctx, migrate)
}

// reconcileMigrateBackup reconcile the backup process during migration.
func (m *MigrateManager) reconcileMigrateBackup(ctx context.Context, migrate *backupapi.Migrate) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch details of the source cluster for migration
	fleetClusters, err := fetchDestinationClusters(ctx, m.Client, migrate.Namespace, migrate.Spec.SourceCluster)
	if err != nil {
		log.Error(err, "Failed to fetch source cluster for migration")
		return ctrl.Result{}, fmt.Errorf("fetching source cluster: %w", err)
	}
	var sourceClusterKey ClusterKey
	var sourceClusterAccess *fleetCluster
	// "migrate.Spec.SourceCluster" must contain one clusters, it is ensured by admission webhook
	for key, value := range fleetClusters {
		sourceClusterKey = key
		sourceClusterAccess = value
	}

	// Construct labels and backup resource details for migration
	migrateLabel := generateVeleroInstanceLabel(MigrateNameLabel, migrate.Name, migrate.Spec.SourceCluster.Fleet)
	sourceBackupName := generateVeleroResourceName(sourceClusterKey.Name, MigrateKind, migrate.Name)
	sourceBackup := buildVeleroBackupInstanceUsingMigrate(&migrate.Spec, migrateLabel, sourceBackupName)

	// Attempt to sync the backup resource
	if err = syncVeleroObj(ctx, sourceClusterKey, sourceClusterAccess, sourceBackup); err != nil {
		log.Error(err, "Failed to create backup resource for migration", "backupName", sourceBackupName)
		return ctrl.Result{}, fmt.Errorf("creating backup resource: %w", err)
	}

	// get the status of Velero restore resources
	veleroBackup := &velerov1.Backup{}
	err = getResourceFromClusterClient(ctx, sourceBackupName, VeleroNamespace, *sourceClusterAccess, veleroBackup)
	if err != nil {
		log.Error(err, "Failed to retrieve backup resource", "backupName", sourceBackupName)
		return ctrl.Result{}, fmt.Errorf("retrieving backup status: %w", err)
	}

	// Update migration status based on the backup details
	currentBackupDetails := &backupapi.BackupDetails{
		ClusterName:           sourceClusterKey.Name,
		ClusterKind:           sourceClusterKey.Kind,
		BackupNameInCluster:   veleroBackup.Name,
		BackupStatusInCluster: &veleroBackup.Status,
	}
	migrate.Status.SourceClusterStatus = currentBackupDetails

	if veleroBackup.Status.Phase == velerov1.BackupPhaseCompleted {
		conditions.MarkTrue(migrate, backupapi.SourceReadyCondition)
	}

	return ctrl.Result{}, nil
}

// reconcileMigrateRestore handles the restore stage of the migration process.
func (m *MigrateManager) reconcileMigrateRestore(ctx context.Context, migrate *backupapi.Migrate) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// If source cluster's backup resource is not ready, return directly.
	if !isMigrateSourceReady(migrate) {
		return ctrl.Result{RequeueAfter: RequeueAfter}, nil
	}

	targetClusters, err := fetchDestinationClusters(ctx, m.Client, migrate.Namespace, migrate.Spec.TargetClusters)
	if err != nil {
		log.Error(err, "Failed to fetch target clusters for migration")
		return ctrl.Result{}, fmt.Errorf("fetching target clusters: %w", err)
	}

	if migrate.Status.Phase != backupapi.MigratePhaseInProgress {
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
		// ensure the velero backup has been sync to current cluster
		referredVeleroBackup := &velerov1.Backup{}
		err := getResourceFromClusterClient(ctx, referredBackupName, VeleroNamespace, *clusterAccess, referredVeleroBackup)
		if apierrors.IsNotFound(err) {
			return ctrl.Result{RequeueAfter: RequeueAfter}, nil
		}

		veleroRestoreName := generateVeleroResourceName(clusterKey.Name, MigrateKind, migrate.Name)
		veleroRestore := buildVeleroRestoreInstanceUsingMigrate(&migrate.Spec, restoreLabel, referredBackupName, veleroRestoreName)
		if err := syncVeleroObj(ctx, clusterKey, clusterAccess, veleroRestore); err != nil {
			log.Error(err, "Failed to creating Velero restore instance", "restoreName", veleroRestoreName)
			return ctrl.Result{}, fmt.Errorf("creating Velero restore instance for cluster %s: %w", clusterKey.Name, err)
		}
	}

	// Collect target clusters backup resource status to current restore
	if migrate.Status.TargetClustersStatus == nil {
		migrate.Status.TargetClustersStatus = []*backupapi.RestoreDetails{}
	}
	migrate.Status.TargetClustersStatus, err = syncVeleroRestoreStatus(ctx, targetClusters, migrate.Status.TargetClustersStatus, MigrateKind, migrate.Name)
	if err != nil {
		log.Error(err, "failed to sync velero restore status for migrate", "migrateName", migrate.Name)
		return ctrl.Result{}, err
	}

	// Determine whether to requeue the reconciliation based on the completion status of all Velero restore resources.
	// If all restore are complete, exit directly without requeuing.
	// Otherwise, requeue the reconciliation after StatusSyncInterval.
	if allRestoreCompleted(migrate.Status.TargetClustersStatus) {
		migrate.Status.Phase = backupapi.MigratePhaseCompleted
		log.Info("Migrate Phase changes", "phase", backupapi.MigratePhaseCompleted)
		return ctrl.Result{}, nil
	} else {
		return ctrl.Result{RequeueAfter: StatusSyncInterval}, nil
	}
}

// reconcileDeleteMigrate handles the deletion process of a Migrate object.
func (m *MigrateManager) reconcileDeleteMigrate(ctx context.Context, migrate *backupapi.Migrate) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch source clusters
	sourceCluster, err := fetchDestinationClusters(ctx, m.Client, migrate.Namespace, migrate.Spec.SourceCluster)
	if err != nil {
		log.Error(err, "failed to fetch source clusters when delete migrate", "migrateName", migrate.Name)
		controllerutil.RemoveFinalizer(migrate, MigrateFinalizer)
		log.Info("Removed finalizer due to fetch destination clusters error", "migrateName", migrate.Name)
		return ctrl.Result{}, err
	}

	// Delete related velero backup instance
	backupList := &velerov1.BackupList{}
	if err := deleteResourcesInClusters(ctx, VeleroNamespace, MigrateNameLabel, migrate.Name, sourceCluster, backupList); err != nil {
		log.Error(err, "failed to delete velero migrate Instances when delete migrate", "restoreName", migrate.Name)
		return ctrl.Result{}, err
	}

	// Fetch target clusters
	targetClusters, err := fetchDestinationClusters(ctx, m.Client, migrate.Namespace, migrate.Spec.TargetClusters)
	if err != nil {
		log.Error(err, "failed to fetch target clusters when delete migrate", "migrateName", migrate.Name)
		controllerutil.RemoveFinalizer(migrate, MigrateFinalizer)
		log.Info("Removed finalizer due to fetch destination clusters error", "migrateName", migrate.Name)
		return ctrl.Result{}, err
	}

	// Delete all related velero restore instance
	restoreList := &velerov1.RestoreList{}
	if err := deleteResourcesInClusters(ctx, VeleroNamespace, MigrateNameLabel, migrate.Name, targetClusters, restoreList); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to delete related %s instances: %w", "migrate", err)
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(migrate, MigrateFinalizer)

	return ctrl.Result{}, nil
}
