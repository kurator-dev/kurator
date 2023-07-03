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
	"strings"

	helmv2b1 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	applicationapi "kurator.dev/kurator/pkg/apis/apps/v1alpha1"
	clusterv1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

// syncPolicyResource synchronizes the sync policy resources for a given application.
func (a *ApplicationManager) syncPolicyResource(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet, syncPolicy *applicationapi.ApplicationSyncPolicy, policyName string) (ctrl.Result, error) {
	policyKind := getSyncPolicyKind(syncPolicy)

	destination := getPolicyDestination(app, syncPolicy)

	// fetch fleet cluster list that recorded in fleet and matches the destination's cluster selector
	fleetClusterList, result, err := a.fetchFleetClusterList(ctx, fleet, destination.ClusterSelector)
	if err != nil || result.RequeueAfter > 0 {
		return result, err
	}
	// Iterate through all clusters, and create/update kustomization/helmRelease for each of them.
	for _, currentFleetCluster := range fleetClusterList {
		// fetch kubeconfig for each cluster.
		kubeconfig := a.generateKubeConfig(currentFleetCluster)

		if result, err1 := a.handleSyncPolicyByKind(ctx, app, policyKind, syncPolicy, policyName, currentFleetCluster, kubeconfig); err1 != nil || result.RequeueAfter > 0 {
			return result, errors.Wrapf(err1, "failed to handleSyncPolicyByKind currentFleetCluster=%s", currentFleetCluster.GetObject().GetName())
		}
	}

	return ctrl.Result{}, nil
}

// fetchFleetClusterList fetch fleet cluster list that recorded in fleet and matches the selector.
func (a *ApplicationManager) fetchFleetClusterList(ctx context.Context, fleet *fleetapi.Fleet, selector *applicationapi.ClusterSelector) ([]ClusterInterface, ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	var fleetClusterList []ClusterInterface

	for _, cluster := range fleet.Spec.Clusters {
		// cluster.kind cluster.name that recorded in fleet must be valid
		kind := cluster.Kind
		name := cluster.Name
		if kind == ClusterKind {
			cluster := &clusterv1alpha1.Cluster{}
			key := client.ObjectKey{
				Name:      name,
				Namespace: fleet.Namespace,
			}
			err := a.Client.Get(ctx, key, cluster)
			if apierrors.IsNotFound(err) {
				return nil, ctrl.Result{RequeueAfter: RequeueAfter}, nil
			}
			if err != nil {
				return nil, ctrl.Result{}, err
			}
			if doLabelsMatchSelector(cluster.Labels, selector) {
				fleetClusterList = append(fleetClusterList, cluster)
			}
		} else if kind == AttachedClusterKind {
			attachedCluster := &clusterv1alpha1.AttachedCluster{}
			key := client.ObjectKey{
				Name:      name,
				Namespace: fleet.Namespace,
			}
			err := a.Client.Get(ctx, key, attachedCluster)
			if apierrors.IsNotFound(err) {
				return nil, ctrl.Result{RequeueAfter: RequeueAfter}, nil
			}
			if err != nil {
				return nil, ctrl.Result{}, err
			}
			if doLabelsMatchSelector(attachedCluster.Labels, selector) {
				fleetClusterList = append(fleetClusterList, attachedCluster)
			}
		} else {
			log.Info("kind of cluster in fleet is not support, skip this cluster", "fleet", fleet.Name, "kind", kind)
		}
	}
	return fleetClusterList, ctrl.Result{}, nil
}

// getKustomizationList returns a list of kustomizations associated with the given application.
func (a *ApplicationManager) getKustomizationList(ctx context.Context, app *applicationapi.Application) (*kustomizev1beta2.KustomizationList, error) {
	kustomizationList := &kustomizev1beta2.KustomizationList{}
	err := a.Client.List(ctx, kustomizationList,
		client.InNamespace(app.Namespace),
		client.MatchingLabels{ApplicationLabel: app.Name})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to fetch kustomizationList for application: %w", err)
	}
	return kustomizationList, nil
}

// getHelmReleaseList returns a list of helmReleases associated with the given application.
func (a *ApplicationManager) getHelmReleaseList(ctx context.Context, app *applicationapi.Application) (*helmv2b1.HelmReleaseList, error) {
	helmReleaseList := &helmv2b1.HelmReleaseList{}
	err := a.Client.List(ctx, helmReleaseList,
		client.InNamespace(app.Namespace),
		client.MatchingLabels{ApplicationLabel: app.Name})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to fetch kustomizationList for application: %w", err)
	}
	return helmReleaseList, nil
}

// handleSyncPolicyByKind handles syncing for a given policy kind (either kustomization or Helm release) based on the provided sync policy.
func (a *ApplicationManager) handleSyncPolicyByKind(
	ctx context.Context,
	app *applicationapi.Application,
	policyKind string,
	syncPolicy *applicationapi.ApplicationSyncPolicy,
	policyName string,
	fleetCluster ClusterInterface,
	kubeConfig *fluxmeta.KubeConfigReference,
) (ctrl.Result, error) {
	policyResourceName := generatePolicyResourceName(policyName, fleetCluster.GetObject().GetObjectKind().GroupVersionKind().Kind, fleetCluster.GetObject().GetName())
	// handle kustomization
	if policyKind == KustomizationKind {
		kustomization := syncPolicy.Kustomization
		// sync kustomization using the provided kubeconfig and source.
		if result, err := a.syncKustomizationForCluster(ctx, app, kustomization, kubeConfig, policyResourceName); err != nil || result.RequeueAfter > 0 {
			return result, err
		}
		return ctrl.Result{}, nil
	}

	// handle helmRelease
	if policyKind == HelmReleaseKind {
		helmRelease := syncPolicy.Helm
		// sync helmRelease using the provided kubeconfig and source.
		if result, err := a.syncHelmReleaseForCluster(ctx, app, helmRelease, kubeConfig, policyResourceName); err != nil || result.RequeueAfter > 0 {
			return result, err
		}
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// generateKubeConfig generates the kubeconfig reference for a cluster within a Fleet.
func (a *ApplicationManager) generateKubeConfig(fleetCluster ClusterInterface) *fluxmeta.KubeConfigReference {
	secretRef := fluxmeta.SecretKeyReference{
		Name: fleetCluster.GetSecretName(),
		Key:  fleetCluster.GetSecretKey(),
	}
	kubeConfig := &fluxmeta.KubeConfigReference{
		SecretRef: secretRef,
	}
	return kubeConfig
}

// syncKustomizationForCluster ensures that the Kustomization object is in sync with Flux's requirements for the object.
func (a *ApplicationManager) syncKustomizationForCluster(ctx context.Context, app *applicationapi.Application, kustomization *applicationapi.Kustomization, kubeConfig *fluxmeta.KubeConfigReference, kustomizationName string) (ctrl.Result, error) {
	// Create a target Kustomization object with details extracted from the provided application's Kustomization spec
	targetKustomization := &kustomizev1beta2.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kustomizationName,
			Namespace: app.Namespace,
			Labels: map[string]string{
				ApplicationLabel: app.Name,
			},
			OwnerReferences: []metav1.OwnerReference{generateApplicationOwnerRef(app)},
		},
		Spec: kustomizev1beta2.KustomizationSpec{
			// Populate the Kustomization spec with information from the provided Kustomization spec
			// Include all relevant details for the Kustomization, like DependsOn, Interval, RetryInterval, KubeConfig, Path, and more.
			DependsOn:     kustomization.DependsOn,
			Interval:      kustomization.Interval,
			RetryInterval: kustomization.RetryInterval,
			KubeConfig:    kubeConfig,
			Path:          kustomization.Path,
			Prune:         kustomization.Prune,
			Patches:       kustomization.Patches,
			Images:        kustomization.Images,
			SourceRef: kustomizev1beta2.CrossNamespaceSourceReference{
				Kind: findSourceKind(app),
				Name: generateSourceName(app),
			},
			Suspend:         kustomization.Suspend,
			TargetNamespace: kustomization.TargetNamespace,
			Timeout:         kustomization.Timeout,
			Force:           kustomization.Force,
			Components:      kustomization.Components,
		},
	}

	// If available, apply Kustomization CommonMetadata data to the target Kustomization
	if kustomization.CommonMetadata != nil {
		targetKustomization.Spec.CommonMetadata = &kustomizev1beta2.CommonMetadata{
			Annotations: kustomization.CommonMetadata.Annotations,
			Labels:      kustomization.CommonMetadata.Labels,
		}
	}

	// Synchronize the target Kustomization object with the Kubernetes API server
	return a.syncResource(ctx, targetKustomization, KustomizationKind)
}

// syncHelmReleaseForCluster ensures that the HelmRelease object is in sync with Flux's requirements for the object.
func (a *ApplicationManager) syncHelmReleaseForCluster(ctx context.Context, app *applicationapi.Application, helmRelease *applicationapi.HelmRelease, kubeConfig *fluxmeta.KubeConfigReference, helmReleaseName string) (ctrl.Result, error) {
	// Create a target HelmRelease object with details extracted from the provided application's HelmRelease spec
	targetHelmRelease := &helmv2b1.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      helmReleaseName,
			Namespace: app.Namespace,
			Labels: map[string]string{
				ApplicationLabel: app.Name,
			},
			OwnerReferences: []metav1.OwnerReference{generateApplicationOwnerRef(app)},
		},
		Spec: helmv2b1.HelmReleaseSpec{
			// Populate the HelmRelease spec with information from the provided HelmRelease spec
			// Include all relevant details for the HelmRelease, like Interval, KubeConfig, Suspend, ReleaseName, and more.
			Interval:           helmRelease.Interval,
			KubeConfig:         kubeConfig,
			Suspend:            helmRelease.Suspend,
			ReleaseName:        helmRelease.ReleaseName,
			TargetNamespace:    helmRelease.TargetNamespace,
			DependsOn:          helmRelease.DependsOn,
			Timeout:            helmRelease.Timeout,
			MaxHistory:         helmRelease.MaxHistory,
			ServiceAccountName: helmRelease.ServiceAccountName,
			PersistentClient:   helmRelease.PersistentClient,
			Install:            helmRelease.Install,
			Upgrade:            helmRelease.Upgrade,
			Rollback:           helmRelease.Rollback,
			Uninstall:          helmRelease.Uninstall,
			ValuesFrom:         helmRelease.ValuesFrom,
			Values:             helmRelease.Values,
		},
	}

	// If available, apply HelmRelease Chart.ObjectMeta data to the target HelmRelease
	if helmRelease.Chart.ObjectMeta != nil {
		targetHelmRelease.Spec.Chart.ObjectMeta = &helmv2b1.HelmChartTemplateObjectMeta{
			Labels:      helmRelease.Chart.ObjectMeta.Labels,
			Annotations: helmRelease.Chart.ObjectMeta.Annotations,
		}
	}

	// Apply the HelmRelease Chart.HelmChartTemplateSpec data to the target HelmRelease
	charSpec := helmRelease.Chart.Spec
	targetHelmRelease.Spec.Chart.Spec = helmv2b1.HelmChartTemplateSpec{
		Chart:   charSpec.Chart,
		Version: charSpec.Version,
		SourceRef: helmv2b1.CrossNamespaceObjectReference{
			Kind: findSourceKind(app),
			Name: generateSourceName(app),
		},
		Interval:          charSpec.Interval,
		ReconcileStrategy: charSpec.ReconcileStrategy,
		ValuesFiles:       charSpec.ValuesFiles,
	}

	// Synchronize the target HelmRelease object with the Kubernetes API server
	return a.syncResource(ctx, targetHelmRelease, HelmReleaseKind)
}

// syncSourceResource synchronizes the source resource based on the application's source specification.
func (a *ApplicationManager) syncSourceResource(ctx context.Context, app *applicationapi.Application) (ctrl.Result, error) {
	kind := findSourceKind(app)
	// Based on the source kind, create the appropriate source object and synchronize it with the Kubernetes API server
	switch kind {
	case GitRepoKind:
		targetSource := &sourcev1beta2.GitRepository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generateSourceName(app),
				Namespace: app.Namespace,
				Labels: map[string]string{
					ApplicationLabel: app.Name,
				},
				OwnerReferences: []metav1.OwnerReference{generateApplicationOwnerRef(app)},
			},
			Spec: *app.Spec.Source.GitRepository,
		}
		return a.syncResource(ctx, targetSource, GitRepoKind)
	case HelmRepoKind:
		targetSource := &sourcev1beta2.HelmRepository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generateSourceName(app),
				Namespace: app.Namespace,
				Labels: map[string]string{
					ApplicationLabel: app.Name,
				},
				OwnerReferences: []metav1.OwnerReference{generateApplicationOwnerRef(app)},
			},
			Spec: *app.Spec.Source.HelmRepository,
		}
		return a.syncResource(ctx, targetSource, HelmRepoKind)
	case OCIRepoKind:
		targetSource := &sourcev1beta2.OCIRepository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generateSourceName(app),
				Namespace: app.Namespace,
				Labels: map[string]string{
					ApplicationLabel: app.Name,
				},
				OwnerReferences: []metav1.OwnerReference{generateApplicationOwnerRef(app)},
			},
			Spec: *app.Spec.Source.OCIRepository,
		}
		return a.syncResource(ctx, targetSource, OCIRepoKind)
	}
	return ctrl.Result{}, nil
}

// createEmptyObject generates an uninitialized instance of the specified resource type.
// This function aids in resource synchronization operations, providing a blank slate for either retrieval or creation.
// If the provided resourceKind isn't recognized, the function returns nil.
func createEmptyObject(resourceKind string) client.Object {
	switch resourceKind {
	case GitRepoKind:
		return &sourcev1beta2.GitRepository{}
	case HelmRepoKind:
		return &sourcev1beta2.HelmRepository{}
	case OCIRepoKind:
		return &sourcev1beta2.OCIRepository{}
	case KustomizationKind:
		return &kustomizev1beta2.Kustomization{}
	case HelmReleaseKind:
		return &helmv2b1.HelmRelease{}
	default:
		return nil
	}
}

// syncResource synchronizes the given `targetSource` resource with the corresponding resource in the Kubernetes API server.
// The resource is identified by its name and namespace, which are obtained from the `targetSource` object.
// If the resource already exists, it is updated with the contents of the `targetSource` object.
// If the resource does not exist, it is created using the contents of the `targetSource` object.
// Returns an error if the synchronization or creation of the resource fails.
func (a *ApplicationManager) syncResource(ctx context.Context, targetSource client.Object, resourceKind string) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// try to get the current resource from the API server
	resourceKey := client.ObjectKey{
		Name:      targetSource.GetName(),
		Namespace: targetSource.GetNamespace(),
	}
	currentResource := createEmptyObject(resourceKind)
	err := a.Client.Get(ctx, resourceKey, currentResource)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, fmt.Sprintf("failed to get %s", resourceKind), resourceKind, resourceKey)
		return ctrl.Result{}, err
	}

	// if not found, create it
	if apierrors.IsNotFound(err) {
		if err := a.Client.Create(ctx, targetSource); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				log.Error(err, fmt.Sprintf("failed to create %s", resourceKind), resourceKind, resourceKey)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// if already exist, update it
	// The following is a type assertion in Go. Type assertion is used here instead of reflection due to its safety and simplicity.
	switch resourceKind {
	case GitRepoKind:
		err = a.updateGitRepository(ctx, currentResource.(*sourcev1beta2.GitRepository), targetSource.(*sourcev1beta2.GitRepository))
	case HelmRepoKind:
		err = a.updateHelmRepository(ctx, currentResource.(*sourcev1beta2.HelmRepository), targetSource.(*sourcev1beta2.HelmRepository))
	case OCIRepoKind:
		err = a.updateOCIRepository(ctx, currentResource.(*sourcev1beta2.OCIRepository), targetSource.(*sourcev1beta2.OCIRepository))
	case KustomizationKind:
		err = a.updateKustomization(ctx, currentResource.(*kustomizev1beta2.Kustomization), targetSource.(*kustomizev1beta2.Kustomization))
	case HelmReleaseKind:
		err = a.updateHelmRelease(ctx, currentResource.(*helmv2b1.HelmRelease), targetSource.(*helmv2b1.HelmRelease))
	default:
		log.Error(err, fmt.Sprintf("resource type %s is not supported", resourceKind))
		return ctrl.Result{}, nil
	}
	// If there is a conflict during the update, it indicates that the resource may have been updated by the Flux controller.
	// In this case, the handler should requeue the resource and wait for the next reconciliation.
	if apierrors.IsConflict(err) {
		return ctrl.Result{RequeueAfter: RequeueAfter}, nil
	}
	if err != nil && !apierrors.IsConflict(err) {
		log.Error(err, fmt.Sprintf("failed to update %s", resourceKind), resourceKind, resourceKey)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// updateGitRepository updates the state of a current GitRepository resource to match the provided target GitRepository resource.
// This function is used by syncResource to keep the actual state of GitRepository resources in sync with the desired state.
func (a *ApplicationManager) updateGitRepository(ctx context.Context, currentResource *sourcev1beta2.GitRepository, targetSource *sourcev1beta2.GitRepository) error {
	currentResource.Spec = targetSource.Spec
	if err := a.Client.Update(ctx, currentResource); err != nil {
		return err
	}
	return nil
}

// updateHelmRepository updates the state of a current HelmRepository resource to match the provided target HelmRepository resource.
// This function is used by syncResource to keep the actual state of HelmRepository resources in sync with the desired state.
func (a *ApplicationManager) updateHelmRepository(ctx context.Context, currentResource *sourcev1beta2.HelmRepository, targetSource *sourcev1beta2.HelmRepository) error {
	currentResource.Spec = targetSource.Spec
	if err := a.Client.Update(ctx, currentResource); err != nil {
		return err
	}
	return nil
}

// updateOCIRepository updates the state of a current OCIRepository resource to match the provided target OCIRepository resource.
// This function is used by syncResource to keep the actual state of OCIRepository resources in sync with the desired state.
func (a *ApplicationManager) updateOCIRepository(ctx context.Context, currentResource *sourcev1beta2.OCIRepository, targetSource *sourcev1beta2.OCIRepository) error {
	currentResource.Spec = targetSource.Spec
	if err := a.Client.Update(ctx, currentResource); err != nil {
		return err
	}
	return nil
}

// updateKustomization updates the state of a current Kustomization resource to match the provided target Kustomization resource.
// This function is used by syncResource to keep the actual state of Kustomization resources in sync with the desired state.
func (a *ApplicationManager) updateKustomization(ctx context.Context, currentResource *kustomizev1beta2.Kustomization, targetSource *kustomizev1beta2.Kustomization) error {
	currentResource.Spec = targetSource.Spec
	if err := a.Client.Update(ctx, currentResource); err != nil {
		return err
	}
	return nil
}

// updateHelmRelease updates the state of a current HelmRelease resource to match the provided target HelmRelease resource.
// This function is used by syncResource to keep the actual state of HelmRelease resources in sync with the desired state.
func (a *ApplicationManager) updateHelmRelease(ctx context.Context, currentResource *helmv2b1.HelmRelease, targetSource *helmv2b1.HelmRelease) error {
	currentResource.Spec = targetSource.Spec
	if err := a.Client.Update(ctx, currentResource); err != nil {
		return err
	}
	return nil
}

// TODO: An application can only have one specified source type. In case of none or multiple source types are specified, should not do these check here, it should be done via validating webhook.

// findSourceKind get the type of the application's source.
func findSourceKind(app *applicationapi.Application) string {
	if app.Spec.Source.GitRepository != nil {
		return GitRepoKind
	}
	if app.Spec.Source.HelmRepository != nil {
		return HelmRepoKind
	}
	if app.Spec.Source.OCIRepository != nil {
		return OCIRepoKind
	}
	return ""
}

// getSyncPolicyKind get the type of the application's syncPolicy.
func getSyncPolicyKind(syncPolicy *applicationapi.ApplicationSyncPolicy) string {
	if syncPolicy.Kustomization != nil {
		return KustomizationKind
	}
	if syncPolicy.Helm != nil {
		return HelmReleaseKind
	}
	return ""
}

// generatePolicyResourceName creates a unique name for a policy resource (such as helmRelease or kustomization)
// based on the provided application, cluster kind, and cluster name.
func generatePolicyResourceName(policyName, clusterKind, clusterName string) string {
	name := policyName + "-" + clusterKind + "-" + clusterName
	name = strings.ToLower(name)
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}

// generateSourceName generates a unique name for the source of an application based on the application's name.
func generateSourceName(app *applicationapi.Application) string {
	name := app.Name

	return name
}

// generateApplicationOwnerRef constructs an OwnerReference object based on the provided application.
func generateApplicationOwnerRef(app *applicationapi.Application) metav1.OwnerReference {
	ownerRef := metav1.OwnerReference{
		APIVersion: applicationapi.GroupVersion.String(),
		Kind:       ApplicationKind,
		Name:       app.Name,
		UID:        app.UID,
	}
	return ownerRef
}

func generatePolicyName(app *applicationapi.Application, index int) string {
	// If no policy name is specified, set a default in the format `<application name>-<index>`.
	if len(app.Spec.SyncPolicies[index].Name) == 0 {
		return app.Name + "-" + strconv.Itoa(index)
	}

	return app.Spec.SyncPolicies[index].Name
}

func generateFleetKey(app *applicationapi.Application) client.ObjectKey {
	var fleetName string
	// if destination of SyncPolicies is not set, we use the destination of application
	if app.Spec.SyncPolicies[0].Destination == nil || len(app.Spec.SyncPolicies[0].Destination.Fleet) == 0 {
		// if destination is not set in both SyncPolicies and application, just return ""
		if app.Spec.Destination == nil {
			fleetName = ""
		} else {
			fleetName = app.Spec.Destination.Fleet
		}
	} else {
		fleetName = app.Spec.SyncPolicies[0].Destination.Fleet
	}
	return client.ObjectKey{
		Namespace: app.Namespace,
		Name:      fleetName,
	}
}

// getPolicyDestination returns the actual destination used by the sync policy.
// The function assumes either Application or its SyncPolicy will have a valid Destination, as this is ensured by the webhook validator.
// If SyncPolicy's Destination is nil, it defaults to Application's Destination.
func getPolicyDestination(app *applicationapi.Application, policy *applicationapi.ApplicationSyncPolicy) applicationapi.ApplicationDestination {
	if policy.Destination == nil {
		return applicationapi.ApplicationDestination{
			Fleet:           app.Spec.Destination.Fleet,
			ClusterSelector: app.Spec.Destination.ClusterSelector,
		}
	}
	return applicationapi.ApplicationDestination{
		Fleet:           policy.Destination.Fleet,
		ClusterSelector: policy.Destination.ClusterSelector,
	}
}

// doLabelsMatchSelector checks if labels match the provided selector.
func doLabelsMatchSelector(labels map[string]string, selector *applicationapi.ClusterSelector) bool {
	// If there is no selector, all labels are considered a match.
	if selector == nil || selector.MatchLabels == nil {
		return true
	}

	for key, value := range selector.MatchLabels {
		if clusterValue, ok := labels[key]; !ok || clusterValue != value {
			// If there is no label for the key,
			// or the label value does not match the selector's value,
			// this labels does not match the selector.
			return false
		}
	}

	return true
}
