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
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	backupapi "kurator.dev/kurator/pkg/apis/backups/v1alpha1"
	fleetmanager "kurator.dev/kurator/pkg/fleet-manager"
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
	log := ctrl.LoggerFrom(ctx).WithValues("migrate", req.NamespacedName)

	migrate := &backupapi.Migrate{}

	if err := m.Client.Get(ctx, req.NamespacedName, migrate); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("migrate does not exist")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	patchHelper, err := patch.NewHelper(migrate, m.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to init patch helper for migrate %s", req.NamespacedName)
	}
	defer func() {
		if err := patchHelper.Patch(ctx, migrate); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch %s  %s", migrate.Name, req.NamespacedName)})
		}
	}()

	if !controllerutil.ContainsFinalizer(migrate, MigrateFinalizer) {
		controllerutil.AddFinalizer(migrate, MigrateFinalizer)
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
		migrate.Status.Phase = backupapi.MigratePhaseBackupInProgress
		log.Info("Migrate Phase changes", "phase", backupapi.MigratePhaseBackupInProgress)
	}

	// The actual migration operation can be divided into two stages
	// 1.the backup stage
	result, err := m.reconcileMigrateBackup(ctx, migrate)
	if err != nil {
		return result, err
	}

	// 2.the restore stage.
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
	var sourceClusterKey fleetmanager.ClusterKey
	var sourceClusterAccess *fleetmanager.FleetCluster
	// "migrate.Spec.SourceCluster" must contain one clusters, it is ensured by admission webhook
	for key, value := range fleetClusters {
		sourceClusterKey = key
		sourceClusterAccess = value
	}

	// Construct Velero backup instance
	migrateLabel := generateVeleroInstanceLabel(MigrateNameLabel, migrate.Name, migrate.Spec.SourceCluster.Fleet)
	sourceBackupName := generateVeleroResourceName(sourceClusterKey.Name, MigrateKind, migrate.Namespace, migrate.Name)
	sourceBackup := buildVeleroBackupFromMigrate(&migrate.Spec, migrateLabel, sourceBackupName)

	// Sync the Velero backup resource
	if err = syncVeleroObj(ctx, sourceClusterAccess, sourceBackup); err != nil {
		log.Error(err, "Failed to create backup resource for migration", "backupName", sourceBackupName)
		return ctrl.Result{}, fmt.Errorf("creating backup resource: %w", err)
	}

	// Get the status of Velero backup resources
	veleroBackup := &velerov1.Backup{}
	err = getResourceFromClusterClient(ctx, sourceBackupName, VeleroNamespace, *sourceClusterAccess, veleroBackup)
	if err != nil {
		log.Error(err, "Failed to retrieve backup resource for migration", "backupName", sourceBackupName)
		return ctrl.Result{}, fmt.Errorf("retrieving backup status: %w", err)
	}

	// Update migration status based on the Velero backup details
	currentBackupDetails := &backupapi.BackupDetails{
		ClusterName:           sourceClusterKey.Name,
		ClusterKind:           sourceClusterKey.Kind,
		BackupNameInCluster:   veleroBackup.Name,
		BackupStatusInCluster: &veleroBackup.Status,
	}
	migrate.Status.SourceClusterStatus = currentBackupDetails

	if veleroBackup.Status.Phase == velerov1.BackupPhaseCompleted {
		conditions.MarkTrue(migrate, backupapi.SourceReadyCondition)
	} else {
		log.Info("Waiting for source backup to be ready", "sourceBackupName", sourceBackupName)
		return ctrl.Result{RequeueAfter: fleetmanager.RequeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

// reconcileMigrateRestore handles the restore stage of the migration process.
func (m *MigrateManager) reconcileMigrateRestore(ctx context.Context, migrate *backupapi.Migrate) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	targetClusters, err := fetchDestinationClusters(ctx, m.Client, migrate.Namespace, migrate.Spec.TargetClusters)
	if err != nil {
		log.Error(err, "Failed to fetch target clusters for migration")
		return ctrl.Result{}, fmt.Errorf("fetching target clusters: %w", err)
	}

	if migrate.Status.Phase != backupapi.MigratePhaseBackupInProgress {
		migrate.Status.Phase = backupapi.MigratePhaseRestoreInProgress
		log.Info("Migrate Phase changes", "phase", backupapi.MigratePhaseRestoreInProgress)
	}

	// referredBackupName is same in different target clusters velero restore, because the velero backup will sync to current cluster.
	// SourceCluster only has one cluster, so the cluster[0].name is the real name of SourceCluster
	referredBackupName := generateVeleroResourceName(migrate.Spec.SourceCluster.Clusters[0].Name, MigrateKind, migrate.Namespace, migrate.Name)
	restoreLabel := generateVeleroInstanceLabel(MigrateNameLabel, migrate.Name, migrate.Spec.TargetClusters.Fleet)
	// Handle create target clusters' velero restore
	var tasks []func() error
	for clusterKey, clusterAccess := range targetClusters {
		// Ensure the velero backup has been sync to current cluster
		referredVeleroBackup := &velerov1.Backup{}
		if err = getResourceFromClusterClient(ctx, referredBackupName, VeleroNamespace, *clusterAccess, referredVeleroBackup); err != nil {
			if apierrors.IsNotFound(err) {
				// if not found, requeue with `RequeueAfter`
				return ctrl.Result{RequeueAfter: fleetmanager.RequeueAfter}, nil
			} else {
				return ctrl.Result{}, err
			}
		}

		veleroRestoreName := generateVeleroResourceName(clusterKey.Name, MigrateKind, migrate.Namespace, migrate.Name)
		veleroRestore := buildVeleroRestoreFromMigrate(&migrate.Spec, restoreLabel, referredBackupName, veleroRestoreName)

		task := newSyncVeleroTaskFunc(ctx, clusterAccess, veleroRestore)
		tasks = append(tasks, task)
	}

	g := &multierror.Group{}
	for _, task := range tasks {
		g.Go(task)
	}

	err = g.Wait().ErrorOrNil()

	if err != nil {
		log.Error(err, "Error encountered during sync velero restore when migrating")
		return ctrl.Result{}, fmt.Errorf("encountered errors during processing: %v", err)
	}

	// Collect target clusters backup resource status to current restore
	if migrate.Status.TargetClustersStatus == nil {
		migrate.Status.TargetClustersStatus = []*backupapi.RestoreDetails{}
	}
	migrate.Status.TargetClustersStatus, err = syncVeleroRestoreStatus(ctx, targetClusters, migrate.Status.TargetClustersStatus, MigrateKind, migrate.Namespace, migrate.Name)
	if err != nil {
		log.Error(err, "failed to sync velero restore status for migrate")
		return ctrl.Result{}, err
	}

	if allRestoreCompleted(migrate.Status.TargetClustersStatus) {
		migrate.Status.Phase = backupapi.MigratePhaseCompleted
		log.Info("Migrate Phase changes", "phase", backupapi.MigratePhaseCompleted)
		return ctrl.Result{}, nil
	} else {
		return ctrl.Result{RequeueAfter: StatusSyncInterval}, nil
	}
}

func (m *MigrateManager) reconcileDeleteMigrate(ctx context.Context, migrate *backupapi.Migrate) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	shouldRemoveFinalizer := false
	defer func() {
		if shouldRemoveFinalizer {
			controllerutil.RemoveFinalizer(migrate, MigrateFinalizer)
			log.Info("Removed finalizer", "migrateName")
		}
	}()

	// Fetch source clusters
	sourceCluster, err := fetchDestinationClusters(ctx, m.Client, migrate.Namespace, migrate.Spec.SourceCluster)
	if err != nil {
		log.Error(err, "Failed to fetch source clusters when delete migrate")
		shouldRemoveFinalizer = true
		return ctrl.Result{}, err
	}

	// Delete related velero backup instance
	backupList := &velerov1.BackupList{}
	err = deleteResourcesInClusters(ctx, VeleroNamespace, MigrateNameLabel, migrate.Name, sourceCluster, backupList)
	if err != nil {
		log.Error(err, "Failed to delete velero backup instances during migrate deletion")
		return ctrl.Result{}, err
	}

	// Fetch target clusters
	targetClusters, err := fetchDestinationClusters(ctx, m.Client, migrate.Namespace, migrate.Spec.TargetClusters)
	if err != nil {
		log.Error(err, "Failed to fetch target clusters when delete migrate")
		shouldRemoveFinalizer = true
		return ctrl.Result{}, err
	}

	// Delete all related velero restore instance
	restoreList := &velerov1.RestoreList{}
	err = deleteResourcesInClusters(ctx, VeleroNamespace, MigrateNameLabel, migrate.Name, targetClusters, restoreList)
	if err != nil {
		log.Error(err, "Failed to delete related velero restore instances during migrate deletion")
		return ctrl.Result{}, err
	}

	shouldRemoveFinalizer = true

	return ctrl.Result{}, nil
}
