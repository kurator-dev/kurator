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

package application

import (
	"context"
	"fmt"

	helmv2b1 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	applicationapi "kurator.dev/kurator/pkg/apis/apps/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	fleetmanager "kurator.dev/kurator/pkg/fleet-manager"
)

const (
	GitRepoKind       = sourcev1beta2.GitRepositoryKind
	HelmRepoKind      = sourcev1beta2.HelmRepositoryKind
	OCIRepoKind       = sourcev1beta2.OCIRepositoryKind
	KustomizationKind = kustomizev1beta2.KustomizationKind
	HelmReleaseKind   = helmv2b1.HelmReleaseKind

	ApplicationLabel     = "apps.kurator.dev/app-name"
	ApplicationKind      = "Application"
	ApplicationFinalizer = "apps.kurator.dev"
)

// ApplicationManager reconciles an Application object
type ApplicationManager struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (a *ApplicationManager) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&applicationapi.Application{}).
		Build(a)
	if err != nil {
		return err
	}

	// Set up watches for the updates to application's status.
	if err := c.Watch(
		source.Kind(mgr.GetCache(), &sourcev1beta2.GitRepository{}),
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for GitRepository: %v", err)
	}

	if err := c.Watch(
		source.Kind(mgr.GetCache(), &sourcev1beta2.HelmRepository{}),
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for HelmRepository: %v", err)
	}

	if err := c.Watch(
		source.Kind(mgr.GetCache(), &sourcev1beta2.OCIRepository{}),
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for OCIRepository: %v", err)
	}

	if err := c.Watch(
		source.Kind(mgr.GetCache(), &kustomizev1beta2.Kustomization{}),
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for Kustomization: %v", err)
	}

	if err := c.Watch(
		source.Kind(mgr.GetCache(), &helmv2b1.HelmRelease{}),
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for HelmRelease: %v", err)
	}

	return nil
}

func (a *ApplicationManager) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx).WithValues("application", req.NamespacedName)

	app := &applicationapi.Application{}
	if err := a.Get(ctx, req.NamespacedName, app); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrapf(err, "failed to get application %s", req.NamespacedName)
	}

	patchHelper, err := patch.NewHelper(app, a.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to init patch helper for application %s", req.NamespacedName)
	}

	defer func() {
		if err := patchHelper.Patch(ctx, app); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch application %s", req.NamespacedName)})
		}
	}()

	// Add finalizer if not exist to void the race condition.
	if !controllerutil.ContainsFinalizer(app, ApplicationFinalizer) {
		controllerutil.AddFinalizer(app, ApplicationFinalizer)
	}

	// there only one fleet, so pre-fetch it here.
	fleetKey := generateFleetKey(app)
	fleet := &fleetapi.Fleet{}
	// Retrieve fleet object based on the defined fleet key
	if err := a.Client.Get(ctx, fleetKey, fleet); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("fleet does not exist", "fleet", fleetKey)
			return ctrl.Result{RequeueAfter: fleetmanager.RequeueAfter}, nil
		}
		// Log error and requeue request if error occurred during fleet retrieval
		log.Error(err, "failed to find fleet", "fleet", fleetKey)
		return ctrl.Result{}, err
	}

	// Handle deletion reconciliation loop.
	if app.DeletionTimestamp != nil {
		return a.reconcileDelete(ctx, app, fleet)
	}

	// Handle normal loop.
	return a.reconcile(ctx, app, fleet)
}

func (a *ApplicationManager) reconcile(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet) (result ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)

	result, err = a.reconcileApplicationResources(ctx, app, fleet)
	if err != nil {
		log.Error(err, "failed to reconcileSyncResources")
	}
	if err != nil || result.RequeueAfter > 0 {
		return
	}

	if result, err = a.reconcileStatus(ctx, app, fleet); err != nil {
		log.Error(err, "failed to reconcile status")
		return ctrl.Result{}, err
	}
	return
}

// reconcileApplicationResources handles the synchronization of resources associated with the current Application resource.
// The associated resources are categorized as 'source' and 'policy'.
// 'source' could be one of gitRepo, helmRepo, or ociRepo while 'policy' can be either kustomizations or helmReleases.
// Any change in Application configuration could potentially lead to creation, deletion, or modification of associated resources in the Kubernetes cluster.
func (a *ApplicationManager) reconcileApplicationResources(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	// Synchronize source resource based on application configuration
	if result, err := a.syncSourceResource(ctx, app); err != nil || result.RequeueAfter > 0 {
		return result, err
	}

	// Iterate over each policy in the application's spec.SyncPolicy
	for index, policy := range app.Spec.SyncPolicies {
		policyName := generatePolicyName(app, index)
		// A policy has a fleet, and a fleet has many clusters. Therefore, a policy may need to create or update multiple kustomizations/helmReleases for each cluster.
		// Synchronize policy resource based on current application, fleet, and policy configuration
		if result, err := a.syncPolicyResource(ctx, app, fleet, policy, policyName); err != nil || result.RequeueAfter > 0 {
			return result, err
		}
	}
	return ctrl.Result{}, nil
}

// reconcileStatus updates the status of resources associated with the current Application resource.
// It does this by fetching the current status of the source (either GitRepoKind or HelmRepoKind) and the sync policy from the API server,
// and updating the Application's status to reflect these current statuses.
func (a *ApplicationManager) reconcileStatus(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet) (result ctrl.Result, err error) {
	if err = a.reconcileSourceStatus(ctx, app); err != nil {
		return ctrl.Result{}, err
	}

	if result, err = a.reconcileSyncStatus(ctx, app, fleet); err != nil {
		return ctrl.Result{}, err
	}

	return
}

// reconcileSourceStatus reconciles the source status of the given application by fetching the status of the source resource (e.g. GitRepository, HelmRepository)
func (a *ApplicationManager) reconcileSourceStatus(ctx context.Context, app *applicationapi.Application) error {
	log := ctrl.LoggerFrom(ctx)

	sourceKey := client.ObjectKey{
		Name:      generateSourceName(app),
		Namespace: app.GetNamespace(),
	}

	if app.Status.SourceStatus == nil {
		app.Status.SourceStatus = &applicationapi.ApplicationSourceStatus{}
	}

	sourceKind := findSourceKind(app)
	// Depending on source kind in application specifications, fetch resource status and update application's source status
	switch sourceKind {
	case GitRepoKind:
		currentResource := &sourcev1beta2.GitRepository{}
		err := a.Client.Get(ctx, sourceKey, currentResource)
		if err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "failed to get GitRepository from the API server when reconciling status")
			return err
		}
		// if not found, return directly. new created GitRepository will be watched in subsequent loop
		if apierrors.IsNotFound(err) {
			return nil
		}
		app.Status.SourceStatus.GitRepoStatus = &currentResource.Status

	case HelmRepoKind:
		currentResource := &sourcev1beta2.HelmRepository{}
		err := a.Client.Get(ctx, sourceKey, currentResource)
		if err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "failed to get HelmRepository from the API server when reconciling status")
			return err
		}
		// if not found, return directly. new created HelmRepository will be watched in subsequent loop
		if apierrors.IsNotFound(err) {
			return nil
		}
		app.Status.SourceStatus.HelmRepoStatus = &currentResource.Status
	}
	return nil
}

// reconcileSyncStatus reconciles the sync status of the given application by finding all Kustomizations and HelmReleases associated with it,
// and updating the sync status of each resource in the application's SyncStatus field.
func (a *ApplicationManager) reconcileSyncStatus(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	var syncStatus []*applicationapi.ApplicationSyncStatus

	// find all kustomization
	kustomizationList, err := a.getKustomizationList(ctx, app)
	if err != nil {
		return ctrl.Result{}, err
	}
	// sync all kustomization status
	for _, kustomization := range kustomizationList.Items {
		kustomizationStatus := &applicationapi.ApplicationSyncStatus{
			Name:                kustomization.Name,
			KustomizationStatus: &kustomization.Status,
		}
		syncStatus = append(syncStatus, kustomizationStatus)
	}

	// find all helmRelease
	helmReleaseList, err := a.getHelmReleaseList(ctx, app)
	if err != nil {
		return ctrl.Result{}, err
	}
	// sync all helmRelease status
	for _, helmRelease := range helmReleaseList.Items {
		helmReleaseStatus := &applicationapi.ApplicationSyncStatus{
			Name:              helmRelease.Name,
			HelmReleaseStatus: &helmRelease.Status,
		}
		syncStatus = append(syncStatus, helmReleaseStatus)
	}

	rolloutStatus := make(map[string]*applicationapi.RolloutStatus)
	// Get rollout status from member clusters
	for index, syncPolicy := range app.Spec.SyncPolicies {
		if syncPolicy.Rollout != nil {
			policyName := generatePolicyName(app, index)
			status, err := a.reconcileRolloutSyncStatus(ctx, app, fleet, syncPolicy, policyName)
			if err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "failed to reconcil rollout status")
			}
			rolloutStatus = mergeMap(status, rolloutStatus)
		}
	}

	// update rollout status
	for index, policyStatus := range syncStatus {
		if _, exist := rolloutStatus[policyStatus.Name]; exist {
			syncStatus[index].RolloutStatus = rolloutStatus[policyStatus.Name]
		}
	}

	app.Status.SyncStatus = syncStatus
	return ctrl.Result{RequeueAfter: StatusSyncInterval}, nil
}

func (a *ApplicationManager) reconcileDelete(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	if err := a.deleteResourcesInMemberClusters(ctx, app, fleet); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to delete rollout resource in cluster")
	}

	controllerutil.RemoveFinalizer(app, ApplicationFinalizer)
	return ctrl.Result{}, nil
}

func (a *ApplicationManager) objectToApplicationFunc(ctx context.Context, o client.Object) []ctrl.Request {
	labels := o.GetLabels()
	if labels[ApplicationLabel] != "" {
		return []ctrl.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: o.GetNamespace(),
					Name:      labels[ApplicationLabel],
				},
			},
		}
	}

	return nil
}
