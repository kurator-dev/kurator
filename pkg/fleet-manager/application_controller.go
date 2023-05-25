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
	"strconv"

	helmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourceapi "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	applicationapi "kurator.dev/kurator/pkg/apis/apps/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

const (
	GitRepoKind       = "GitRepository"
	HelmRepoKind      = "HelmRepository"
	OCIRepoKind       = "OCIRepository"
	KustomizationKind = "Kustomization"
	HelmReleaseKind   = "HelmRelease"

	ApplicationLabel     = "apps.kurator.dev/app-name"
	AppsPolicyLabel      = "apps.kurator.dev/policy-name"
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

	// TODO: Consider whether it's necessary to watch the fleet/cluster/attachedCluster.
	//  For example, if a cluster is added or removed from the fleet, should the application respond correspondingly?
	//  Note that the relationship between fleet/cluster/attachedCluster and application is single-to-many, which can be complex to implement.
	if err := c.Watch(
		&source.Kind{Type: &fleetapi.Fleet{}},
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for GitRepository: %v", err)
	}

	// Set up watches for the updates to application's status.
	// GitRepository/HelmRepository/Kustomization/HelmRelease and application have a many-to-single relationship.
	if err := c.Watch(
		&source.Kind{Type: &sourcev1.GitRepository{}},
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for GitRepository: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &sourceapi.HelmRepository{}},
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for HelmRepository: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kustomizev1.Kustomization{}},
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for Kustomization: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &helmv2beta1.HelmRelease{}},
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for HelmRelease: %v", err)
	}

	return nil
}

func (a *ApplicationManager) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	app := &applicationapi.Application{}
	if err := a.Get(ctx, req.NamespacedName, app); err != nil {
		if apiserrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrapf(err, "failed to get application %s", req.NamespacedName)
	}

	log := ctrl.LoggerFrom(ctx)
	log = log.WithValues("application", klog.KObj(app))

	patchHelper, err := patch.NewHelper(app, a.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to init patch helper for application %s", req.NamespacedName)
	}

	defer func() {
		patchOpts := []patch.Option{}
		if err := patchHelper.Patch(ctx, app, patchOpts...); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch application %s", req.NamespacedName)})
		}
	}()

	// Add finalizer if not exist to void the race condition.
	if !controllerutil.ContainsFinalizer(app, ApplicationFinalizer) {
		controllerutil.AddFinalizer(app, ApplicationFinalizer)
		return ctrl.Result{}, nil
	}

	// Define object key for fleet based on the current policy's destination
	fleetKey := client.ObjectKey{
		Namespace: app.Namespace,
		// there only one fleet
		Name: app.Spec.SyncPolicy[0].Destination.Fleet,
	}
	fleet := &fleetapi.Fleet{}
	// Retrieve fleet object based on the defined fleet key
	if err := a.Client.Get(ctx, fleetKey, fleet); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("fleet does not exist", "fleet", fleetKey)
			return ctrl.Result{RequeueAfter: RequeueAfter}, nil
		}

		// Log error and requeue request if error occurred during fleet retrieval
		log.Error(err, "failed to find fleet", "fleet", fleetKey)
		return ctrl.Result{}, err
	}

	// label the fleet
	if fleet.GetLabels() == nil {
		fleet.SetLabels(make(map[string]string))
	}
	// todo: there is a bug here if more than one application refer the same fleet: the new one will override the old
	//  but it is needed to reconcile application when the fleet' cluster is all joined
	if fleet.GetLabels()[ApplicationLabel] != app.Name {
		labels := fleet.GetLabels()
		labels[ApplicationLabel] = app.Name
		fleet.SetLabels(labels)
		err := a.Update(ctx, fleet)
		if err != nil {
			log.Error(err, "unable to label fleet", "fleet", fleet.Name)
			return ctrl.Result{}, err
		}
	}

	// Handle deletion reconciliation loop.
	if app.DeletionTimestamp != nil {
		return a.reconcileDelete(ctx, app, fleet)
	}

	// Handle normal loop.
	return a.reconcile(ctx, app, fleet)
}

func (a *ApplicationManager) reconcile(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if result, err := a.reconcileInitializeParameters(ctx, app); err != nil || result.RequeueAfter > 0 {
		log.Error(err, "failed to reconcileInitializeParameters")
		return result, err
	}

	if result, err := a.reconcileSyncResources(ctx, app, fleet); err != nil || result.RequeueAfter > 0 {
		log.Error(err, "failed to reconcileSyncResources")

		return result, err
	}

	if result, err := a.reconcileSyncStatus(ctx, app); err != nil || result.RequeueAfter > 0 {
		log.Error(err, "failed to reconcileSyncStatus")

		return result, err
	}

	return ctrl.Result{}, nil
}

// reconcileInitializeParameters initializes the parameters for the given application. It also ensures some parameters is valid.
func (a *ApplicationManager) reconcileInitializeParameters(ctx context.Context, app *applicationapi.Application) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Find the source kind for the given application. Return an error if no source or more than one source is specified.
	sourceKind, err := findSourceKind(app)
	if err != nil {
		log.Error(err, "failed to find source kind")
		return ctrl.Result{}, err
	}

	// Set the source kind for the application.
	app.Spec.Source.Kind = sourceKind

	// Iterate over all sync policies.
	for index, policy := range app.Spec.SyncPolicy {
		// If no policy name is specified, set a default in the format `<application name>-<index>`.
		if len(policy.Name) == 0 {
			policy.Name = app.Name + "-" + strconv.Itoa(index)
		}

		// Currently, only two pairs of source types are supported: 'gitRepo + kustomization' and 'helmRepo + helmRelease'.
		if sourceKind == GitRepoKind && policy.Kustomization == nil {
			log.Error(fmt.Errorf("source kind is GitRepoKind, but policy.Kustomization is nil"), "policyName", policy.Name)
			return ctrl.Result{}, nil
		}

		if sourceKind == HelmRepoKind && policy.Helm == nil {
			log.Error(fmt.Errorf("source kind is HelmRepoKind, but policy.policy.Helm is nil"), "policyName", policy.Name)
			return ctrl.Result{}, nil
		}
	}

	// If the application's source status is nil, set it to an empty ApplicationSourceStatus object.
	if app.Status.SourceStatus == nil {
		app.Status.SourceStatus = &applicationapi.ApplicationSourceStatus{}
	}

	return ctrl.Result{}, nil
}

// reconcileSyncResources handles the synchronization of resources associated with the current Application resource.
// The associated resources are categorized as 'source' and 'policy'.
// 'source' could be one of gitRepo, helmRepo, or ociRepo while 'policy' can be either kustomizations or helmReleases.
// Any change in Application configuration could potentially lead to creation, deletion, or modification of associated resources in the Kubernetes cluster.
func (a *ApplicationManager) reconcileSyncResources(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	// Synchronize source resource based on application configuration
	if err := a.syncSourceResource(ctx, app); err != nil {
		return ctrl.Result{}, err
	}

	// todo: if we need to delete the kustomization/helmReleases when the cluster is removed from the fleet?
	// todo: if we need to delete the kustomization/helmReleases when the fleet is removed from the application?

	// Iterate over each policy in the application's spec.SyncPolicy
	for _, policy := range app.Spec.SyncPolicy {
		// A policy has a fleet, and a fleet has many clusters. Therefore, a policy may need to create or update multiple kustomizations/helmReleases for each cluster.
		// Synchronize policy resource based on current application, fleet, and policy configuration
		if err := a.SyncPolicyResource(ctx, app, fleet, policy); err != nil {
			log.Error(err, "failed to sync policy resource", "fleet", fleet.Name)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// reconcileSyncStatus updates the status of resources associated with the current Application resource.
// It does this by fetching the current status of the source (either GitRepoKind or HelmRepoKind) and the sync policy from the API server,
// and updating the Application's status to reflect these current statuses.
func (a *ApplicationManager) reconcileSyncStatus(ctx context.Context, app *applicationapi.Application) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// reconcile Sync source Status
	sourceKey := client.ObjectKey{
		Name:      generateSourceName(app),
		Namespace: app.GetNamespace(),
	}

	// Depending on source kind in application specifications, fetch resource status and update application's source status
	switch app.Spec.Source.Kind {
	case GitRepoKind:
		currentResource := &sourcev1.GitRepository{}
		if err := a.Client.Get(ctx, sourceKey, currentResource); err != nil {
			log.Error(err, "failed to get GitRepository from the API server when reconciling status")
			return ctrl.Result{}, nil
		}
		app.Status.SourceStatus.GitRepoStatus = &currentResource.Status

	case HelmRepoKind:
		currentResource := &sourceapi.HelmRepository{}
		if err := a.Client.Get(ctx, sourceKey, currentResource); err != nil {
			log.Error(err, "failed to get HelmRepository from the API server when reconciling status")
			return ctrl.Result{}, err
		}
		app.Status.SourceStatus.HelmRepoStatus = &currentResource.Status
	}

	// Depending on source kind in application specifications, fetch associated resources and update application's sync status
	switch app.Spec.Source.Kind {
	case GitRepoKind:
		var kustomizationList kustomizev1.KustomizationList
		if err := a.Client.List(ctx, &kustomizationList, client.InNamespace(app.Namespace), client.MatchingLabels{ApplicationLabel: app.Name}); err != nil {
			return ctrl.Result{}, err
		}

		var syncStatus []*applicationapi.ApplicationSyncStatus
		for _, kustomization := range kustomizationList.Items {
			currentStatus := &applicationapi.ApplicationSyncStatus{
				Name:                kustomization.Name,
				KustomizationStatus: &kustomization.Status,
			}
			syncStatus = append(syncStatus, currentStatus)
		}
		app.Status.SyncStatus = syncStatus

	case HelmRepoKind:
		var helmReleaseList helmv2beta1.HelmReleaseList
		if err := a.Client.List(ctx, &helmReleaseList, client.InNamespace(app.Namespace), client.MatchingLabels{ApplicationLabel: app.Name}); err != nil {
			return ctrl.Result{}, err
		}

		var syncStatus []*applicationapi.ApplicationSyncStatus
		for _, helmRelease := range helmReleaseList.Items {
			currentStatus := &applicationapi.ApplicationSyncStatus{
				Name:              helmRelease.Name,
				HelmReleaseStatus: &helmRelease.Status,
			}
			syncStatus = append(syncStatus, currentStatus)
		}
		app.Status.SyncStatus = syncStatus
	}

	return ctrl.Result{}, nil
}

func (a *ApplicationManager) reconcileDelete(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// remove application label on fleet
	if fleet.GetLabels()[FleetLabel] == app.Name {
		delete(fleet.GetLabels(), FleetLabel)
		err := a.Update(ctx, fleet)
		if err != nil {
			log.Error(err, "unable to remove application label", "fleet", fleet.GetName())
			return ctrl.Result{}, nil
		}
	}

	controllerutil.RemoveFinalizer(app, ApplicationFinalizer)

	return ctrl.Result{}, nil
}

func (a *ApplicationManager) objectToApplicationFunc(o client.Object) []ctrl.Request {
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
