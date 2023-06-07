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

	helmv2b1 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	clusterv1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
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

var fleetToApplicationMap = map[string][]string{}
var clusterToApplicationMap = map[string][]string{}
var attachedClusterToApplicationMap = map[string][]string{}

// SetupWithManager sets up the controller with the Manager.
func (a *ApplicationManager) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&applicationapi.Application{}).
		Build(a)
	if err != nil {
		return err
	}

	// Set up watches for the updates to application's resource.
	if err := c.Watch(
		&source.Kind{Type: &fleetapi.Fleet{}},
		handler.EnqueueRequestsFromMapFunc(a.fleetToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for Fleet: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &clusterv1alpha1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(a.clusterToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for Cluster: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &clusterv1alpha1.AttachedCluster{}},
		handler.EnqueueRequestsFromMapFunc(a.attachedClusterToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for AttachedCluster: %v", err)
	}

	// Set up watches for the updates to application's status.
	if err := c.Watch(
		&source.Kind{Type: &sourcev1beta2.GitRepository{}},
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for GitRepository: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &sourcev1beta2.HelmRepository{}},
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for HelmRepository: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &sourcev1beta2.OCIRepository{}},
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for OCIRepository: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kustomizev1beta2.Kustomization{}},
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for Kustomization: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &helmv2b1.HelmRelease{}},
		handler.EnqueueRequestsFromMapFunc(a.objectToApplicationFunc),
	); err != nil {
		return fmt.Errorf("failed to add a Watch for HelmRelease: %v", err)
	}

	return nil
}

func (a *ApplicationManager) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	app := &applicationapi.Application{}
	if err := a.Get(ctx, req.NamespacedName, app); err != nil {
		if apierrors.IsNotFound(err) {
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

	var fleetName string
	if app.Spec.Destination != nil {
		fleetName = app.Spec.Destination.Fleet
	} else {
		fleetName = app.Spec.SyncPolicies[0].Destination.Fleet
	}
	// there only one fleet, so pre-fetch it here.
	fleetKey := client.ObjectKey{
		Namespace: app.Namespace,
		Name:      fleetName,
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

	// Add this relation to fleetToApplicationMap
	if fleetToApplicationMap[fleet.Name] == nil {
		fleetToApplicationMap[fleet.Name] = make([]string, 0)
	}
	fleetToApplicationMap[fleet.Name] = append(fleetToApplicationMap[fleet.Name], app.Name)

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

	if result, err := a.reconcileApplicationResources(ctx, app, fleet); err != nil || result.RequeueAfter > 0 {
		log.Error(err, "failed to reconcileSyncResources")

		return result, err
	}

	if result, err := a.reconcileStatus(ctx, app); err != nil || result.RequeueAfter > 0 {
		log.Error(err, "failed to reconcileSyncStatus")

		return result, err
	}

	return ctrl.Result{}, nil
}

// reconcileInitializeParameters initializes the parameters for the given application.
func (a *ApplicationManager) reconcileInitializeParameters(ctx context.Context, app *applicationapi.Application) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	sourceKind := findSourceKind(app)

	// Iterate over all sync policies.
	for _, policy := range app.Spec.SyncPolicies {
		// Currently, only two pairs of source types are supported: 'gitRepo + kustomization' and 'helmRepo + helmRelease'.
		// TODO: support other pairs
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
func (a *ApplicationManager) reconcileApplicationResources(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	// Synchronize source resource based on application configuration
	if err := a.syncSourceResource(ctx, app); err != nil {
		return ctrl.Result{}, err
	}

	// todo: if we need to delete the kustomization/helmReleases when the cluster is removed from the fleet?
	// todo: if we need to delete the kustomization/helmReleases when the fleet is removed from the application?

	// Iterate over each policy in the application's spec.SyncPolicy
	for index, policy := range app.Spec.SyncPolicies {
		policyName := generatePolicyName(app, index)
		// A policy has a fleet, and a fleet has many clusters. Therefore, a policy may need to create or update multiple kustomizations/helmReleases for each cluster.
		// Synchronize policy resource based on current application, fleet, and policy configuration
		if err := a.syncPolicyResource(ctx, app, fleet, policy, policyName); err != nil {
			log.Error(err, "failed to sync policy resource", "fleet", fleet.Name)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// reconcileSyncStatus updates the status of resources associated with the current Application resource.
// It does this by fetching the current status of the source (either GitRepoKind or HelmRepoKind) and the sync policy from the API server,
// and updating the Application's status to reflect these current statuses.
func (a *ApplicationManager) reconcileStatus(ctx context.Context, app *applicationapi.Application) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// reconcile Sync source Status
	sourceKey := client.ObjectKey{
		Name:      generateSourceName(app),
		Namespace: app.GetNamespace(),
	}

	sourceKind := findSourceKind(app)
	// Depending on source kind in application specifications, fetch resource status and update application's source status
	switch sourceKind {
	case GitRepoKind:
		currentResource := &sourcev1beta2.GitRepository{}
		if err := a.Client.Get(ctx, sourceKey, currentResource); err != nil {
			log.Error(err, "failed to get GitRepository from the API server when reconciling status")
			return ctrl.Result{}, nil
		}
		app.Status.SourceStatus.GitRepoStatus = &currentResource.Status

	case HelmRepoKind:
		currentResource := &sourcev1beta2.HelmRepository{}
		if err := a.Client.Get(ctx, sourceKey, currentResource); err != nil {
			log.Error(err, "failed to get HelmRepository from the API server when reconciling status")
			return ctrl.Result{}, err
		}
		app.Status.SourceStatus.HelmRepoStatus = &currentResource.Status
	}

	// Depending on source kind in application specifications, fetch associated resources and update application's sync status
	switch sourceKind {
	case GitRepoKind:
		var kustomizationList kustomizev1beta2.KustomizationList
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
		var helmReleaseList helmv2b1.HelmReleaseList
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

	if _, ok := fleetToApplicationMap[fleet.Name]; !ok {
		log.Info("current fleet is not recorded in fleetToApplicationMap")
	} else {
		// remove current mapping of fleetToApplicationMap
		applications := fleetToApplicationMap[fleet.Name]
		for i, application := range applications {
			if application == app.Name {
				// delete it
				fleetToApplicationMap[fleet.Name] = append(applications[:i], applications[i+1:]...)
			}
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

func (a *ApplicationManager) fleetToApplicationFunc(o client.Object) []ctrl.Request {
	c, ok := o.(*fleetapi.Fleet)
	if !ok {
		panic(fmt.Sprintf("Expected a Fleet but got a %T", o))
	}
	var result []ctrl.Request

	applicationNames, ok := fleetToApplicationMap[c.Name]

	if ok {
		for _, applicationName := range applicationNames {
			result = append(result, ctrl.Request{NamespacedName: client.ObjectKey{Namespace: c.GetNamespace(), Name: applicationName}})
		}
	}

	return result
}

func (a *ApplicationManager) clusterToApplicationFunc(o client.Object) []ctrl.Request {
	c, ok := o.(*clusterv1alpha1.Cluster)
	if !ok {
		panic(fmt.Sprintf("Expected a Fleet but got a %T", o))
	}
	var result []ctrl.Request

	applicationNames, ok := clusterToApplicationMap[c.Name]

	if ok {
		for _, applicationName := range applicationNames {
			result = append(result, ctrl.Request{NamespacedName: client.ObjectKey{Namespace: c.GetNamespace(), Name: applicationName}})
		}
	}

	return result
}

func (a *ApplicationManager) attachedClusterToApplicationFunc(o client.Object) []ctrl.Request {
	c, ok := o.(*clusterv1alpha1.AttachedCluster)
	if !ok {
		panic(fmt.Sprintf("Expected a Fleet but got a %T", o))
	}
	var result []ctrl.Request

	applicationNames, ok := attachedClusterToApplicationMap[c.Name]

	if ok {
		for _, applicationName := range applicationNames {
			result = append(result, ctrl.Request{NamespacedName: client.ObjectKey{Namespace: c.GetNamespace(), Name: applicationName}})
		}
	}

	return result
}
