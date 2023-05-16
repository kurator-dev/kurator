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
	"strings"

	helmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourceapi "github.com/fluxcd/source-controller/api/v1beta2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	applicationapi "kurator.dev/kurator/pkg/apis/apps/v1alpha1"
	clusterv1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

// SyncPolicyResource synchronizes the sync policy resources for a given application.
func (a *ApplicationManager) SyncPolicyResource(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet, syncPolicy *applicationapi.ApplicationSyncPolicy) error {
	log := ctrl.LoggerFrom(ctx)
	sourceKind := app.Spec.Source.Kind

	// Merge the list of clusters and attached clusters into a single list.
	fleetClusterList, err := a.mergeClusterLists(ctx, fleet)
	if err != nil {
		return err
	}

	if len(fleetClusterList) == 0 {
		log.Info("no cluster is found in current fleet", "fleet", fleet.Name)
		return nil
	}

	// Handle each cluster in current fleet.
	for _, currentFleetCluster := range fleetClusterList {
		// fetch kubeconfig for each cluster.
		kubeConfig := a.generateKubeConfig(currentFleetCluster)

		// handle gitRepo + kustomization
		if sourceKind == GitRepoKind {
			kustomization := syncPolicy.Kustomization
			kustomizationName := generateKustomizationName(app, syncPolicy, currentFleetCluster.GetObject().GetObjectKind().GroupVersionKind().Kind, currentFleetCluster.GetObject().GetName())

			// create flux kustomization using kubeconfig and source.
			if err := a.syncKustomizationForCluster(ctx, app, kustomization, kubeConfig, kustomizationName); err != nil {
				log.Error(err, "failed to syncKustomizationForCluster", "kustomizationName", kustomizationName)
				return err
			}
			return nil
		}

		// handle helmRepo + helmRelease
		if sourceKind == HelmRepoKind {
			helmRelease := syncPolicy.Helm
			helmReleaseName := generateHelmReleaseName(app, syncPolicy, currentFleetCluster.GetObject().GetObjectKind().GroupVersionKind().Kind, currentFleetCluster.GetObject().GetName())

			// create flux helmRelease using kubeconfig and source.
			if err := a.syncHelmReleaseForCluster(ctx, app, helmRelease, kubeConfig, helmReleaseName); err != nil {
				log.Error(err, "failed to syncHelmReleaseForCluster", "helmReleaseName", helmReleaseName)
				return err
			}
			return nil
		}

		// todo: what if kind is ociRepo
		if sourceKind != GitRepoKind && sourceKind != HelmRepoKind {
			return fmt.Errorf("current source kind is %s, this kind is unsupported", sourceKind)
		}
	}
	return nil
}

// mergeClusterLists merges the lists of Clusters and AttachedClusters associated with the specified Fleet.
func (a *ApplicationManager) mergeClusterLists(ctx context.Context, fleet *fleetapi.Fleet) ([]ClusterInterface, error) {
	log := ctrl.LoggerFrom(ctx)

	var clusterList clusterv1alpha1.ClusterList
	var attachedClusterList clusterv1alpha1.AttachedClusterList

	if err := a.Client.List(ctx, &clusterList,
		client.InNamespace(fleet.Namespace),
		client.MatchingLabels{FleetLabel: fleet.Name}); err != nil {
		log.Error(err, "failed to fetch clusterList for fleet", "fleet", fleet.Name)
		return nil, err
	}

	if err := a.Client.List(ctx, &attachedClusterList,
		client.InNamespace(fleet.Namespace),
		client.MatchingLabels{FleetLabel: fleet.Name}); err != nil {
		log.Error(err, "failed to fetch attachedClusterList for fleet", "fleet", fleet.Name)
		return nil, err
	}

	var fleetClusterList []ClusterInterface
	for _, cluster := range clusterList.Items {
		fleetClusterList = append(fleetClusterList, &cluster)
	}
	for _, attachedCluster := range attachedClusterList.Items {
		fleetClusterList = append(fleetClusterList, &attachedCluster)
	}
	return fleetClusterList, nil
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
func (a *ApplicationManager) syncKustomizationForCluster(ctx context.Context, app *applicationapi.Application, kustomization *applicationapi.Kustomization, kubeConfig *fluxmeta.KubeConfigReference, kustomizationName string) error {
	// Create a target Kustomization object with details extracted from the provided application's Kustomization spec
	targetKustomization := &kustomizev1.Kustomization{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kustomizationName,
			Namespace: app.Namespace,
			Labels: map[string]string{
				ApplicationLabel: app.Name,
			},
			OwnerReferences: []metav1.OwnerReference{generateApplicationOwnerRef(app)},
		},
		Spec: kustomizev1.KustomizationSpec{
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
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: GitRepoKind,
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
		targetKustomization.Spec.CommonMetadata = &kustomizev1.CommonMetadata{
			Annotations: kustomization.CommonMetadata.Annotations,
			Labels:      kustomization.CommonMetadata.Labels,
		}
	}

	// Synchronize the target Kustomization object with the Kubernetes API server
	return a.syncResource(ctx, targetKustomization, KustomizationKind)
}

// syncHelmReleaseForCluster ensures that the HelmRelease object is in sync with Flux's requirements for the object.
func (a *ApplicationManager) syncHelmReleaseForCluster(ctx context.Context, app *applicationapi.Application, helmRelease *applicationapi.HelmRelease, kubeConfig *fluxmeta.KubeConfigReference, kustomizationName string) error {
	// Create a target HelmRelease object with details extracted from the provided application's HelmRelease spec
	targetHelmRelease := &helmv2beta1.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kustomizationName,
			Namespace: app.Namespace,
			Labels: map[string]string{
				ApplicationLabel: app.Name,
			},
			OwnerReferences: []metav1.OwnerReference{generateApplicationOwnerRef(app)},
		},
		Spec: helmv2beta1.HelmReleaseSpec{
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
		targetHelmRelease.Spec.Chart.ObjectMeta = &helmv2beta1.HelmChartTemplateObjectMeta{
			Labels:      helmRelease.Chart.ObjectMeta.Labels,
			Annotations: helmRelease.Chart.ObjectMeta.Annotations,
		}
	}

	// Apply the HelmRelease Chart.HelmChartTemplateSpec data to the target HelmRelease
	charSpec := helmRelease.Chart.Spec
	targetHelmRelease.Spec.Chart.Spec = helmv2beta1.HelmChartTemplateSpec{
		Chart:   charSpec.Chart,
		Version: charSpec.Version,
		SourceRef: helmv2beta1.CrossNamespaceObjectReference{
			Kind: HelmRepoKind,
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
func (a *ApplicationManager) syncSourceResource(ctx context.Context, app *applicationapi.Application) error {
	kind := app.Spec.Source.Kind
	// Based on the source kind, create the appropriate source object and synchronize it with the Kubernetes API server
	switch kind {
	case GitRepoKind:
		targetSource := &sourcev1.GitRepository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generateSourceName(app),
				Namespace: app.Namespace,
				Labels: map[string]string{
					ApplicationLabel: app.Name,
				},
				OwnerReferences: []metav1.OwnerReference{generateApplicationOwnerRef(app)},
			},
			Spec: *app.Spec.Source.GitRepo,
		}
		return a.syncResource(ctx, targetSource, GitRepoKind)
	case HelmRepoKind:
		targetSource := &sourceapi.HelmRepository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generateSourceName(app),
				Namespace: app.Namespace,
				Labels: map[string]string{
					ApplicationLabel: app.Name,
				},
				OwnerReferences: []metav1.OwnerReference{generateApplicationOwnerRef(app)},
			},
			Spec: *app.Spec.Source.HelmRepo,
		}
		return a.syncResource(ctx, targetSource, HelmRepoKind)
	case OCIRepoKind:
		targetSource := &sourceapi.OCIRepository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generateSourceName(app),
				Namespace: app.Namespace,
				Labels: map[string]string{
					ApplicationLabel: app.Name,
				},
				OwnerReferences: []metav1.OwnerReference{generateApplicationOwnerRef(app)},
			},
			Spec: *app.Spec.Source.OCIRepo,
		}
		return a.syncResource(ctx, targetSource, OCIRepoKind)
	}
	return nil
}

// createEmptyObject generates an uninitialized instance of the specified resource type.
// This function aids in resource synchronization operations, providing a blank slate for either retrieval or creation.
// If the provided resourceKind isn't recognized, the function returns nil.
func createEmptyObject(resourceKind string) client.Object {
	switch resourceKind {
	case GitRepoKind:
		return &sourcev1.GitRepository{}
	case HelmRepoKind:
		return &sourceapi.HelmRepository{}
	case OCIRepoKind:
		return &sourceapi.OCIRepository{}
	case KustomizationKind:
		return &kustomizev1.Kustomization{}
	case HelmReleaseKind:
		return &helmv2beta1.HelmRelease{}
	default:
		return nil
	}
}

// syncResource synchronizes the given `targetSource` resource with the corresponding resource in the Kubernetes API server.
// The resource is identified by its name and namespace, which are obtained from the `targetSource` object.
// If the resource already exists, it is updated with the contents of the `targetSource` object.
// If the resource does not exist, it is created using the contents of the `targetSource` object.
// Returns an error if the synchronization or creation of the resource fails.
func (a *ApplicationManager) syncResource(ctx context.Context, targetSource client.Object, resourceKind string) error {
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
		return err
	}

	// if not found, create it
	if apierrors.IsNotFound(err) {
		if err := a.Client.Create(ctx, targetSource); err != nil {
			log.Error(err, fmt.Sprintf("failed to get %s", resourceKind), resourceKind, resourceKey)
			return err
		}
		log.Info(fmt.Sprintf("create %s successful", resourceKind), resourceKind, resourceKey)
		return nil
	}

	// if already exist, update it
	// The following is a type assertion in Go. Type assertion is used here instead of reflection due to its safety and simplicity.
	switch resourceKind {
	case GitRepoKind:
		err = a.updateGitRepository(ctx, currentResource.(*sourcev1.GitRepository), targetSource.(*sourcev1.GitRepository))
	case HelmRepoKind:
		err = a.updateHelmRepository(ctx, currentResource.(*sourceapi.HelmRepository), targetSource.(*sourceapi.HelmRepository))
	case OCIRepoKind:
		err = a.updateOCIRepository(ctx, currentResource.(*sourceapi.OCIRepository), targetSource.(*sourceapi.OCIRepository))
	case KustomizationKind:
		err = a.updateKustomization(ctx, currentResource.(*kustomizev1.Kustomization), targetSource.(*kustomizev1.Kustomization))
	case HelmReleaseKind:
		err = a.updateHelmRelease(ctx, currentResource.(*helmv2beta1.HelmRelease), targetSource.(*helmv2beta1.HelmRelease))
	default:
		log.Error(err, fmt.Sprintf("resource type %s is not supported", resourceKind))
		return nil
	}
	if err != nil {
		log.Error(err, fmt.Sprintf("failed to update %s", resourceKind), resourceKind, resourceKey)
		return err
	}

	return nil
}

// updateGitRepository updates the state of a current GitRepository resource to match the provided target GitRepository resource.
// This function is used by syncResource to keep the actual state of GitRepository resources in sync with the desired state.
func (a *ApplicationManager) updateGitRepository(ctx context.Context, currentResource *sourcev1.GitRepository, targetSource *sourcev1.GitRepository) error {
	currentResource.Spec = targetSource.Spec
	if err := a.Client.Update(ctx, currentResource); err != nil {
		return err
	}
	return nil
}

// updateHelmRepository updates the state of a current HelmRepository resource to match the provided target HelmRepository resource.
// This function is used by syncResource to keep the actual state of HelmRepository resources in sync with the desired state.
func (a *ApplicationManager) updateHelmRepository(ctx context.Context, currentResource *sourceapi.HelmRepository, targetSource *sourceapi.HelmRepository) error {
	currentResource.Spec = targetSource.Spec
	if err := a.Client.Update(ctx, currentResource); err != nil {
		return err
	}
	return nil
}

// updateOCIRepository updates the state of a current OCIRepository resource to match the provided target OCIRepository resource.
// This function is used by syncResource to keep the actual state of OCIRepository resources in sync with the desired state.
func (a *ApplicationManager) updateOCIRepository(ctx context.Context, currentResource *sourceapi.OCIRepository, targetSource *sourceapi.OCIRepository) error {
	currentResource.Spec = targetSource.Spec
	if err := a.Client.Update(ctx, currentResource); err != nil {
		return err
	}
	return nil
}

// updateKustomization updates the state of a current Kustomization resource to match the provided target Kustomization resource.
// This function is used by syncResource to keep the actual state of Kustomization resources in sync with the desired state.
func (a *ApplicationManager) updateKustomization(ctx context.Context, currentResource *kustomizev1.Kustomization, targetSource *kustomizev1.Kustomization) error {
	currentResource.Spec = targetSource.Spec
	if err := a.Client.Update(ctx, currentResource); err != nil {
		return err
	}
	return nil
}

// updateHelmRelease updates the state of a current HelmRelease resource to match the provided target HelmRelease resource.
// This function is used by syncResource to keep the actual state of HelmRelease resources in sync with the desired state.
func (a *ApplicationManager) updateHelmRelease(ctx context.Context, currentResource *helmv2beta1.HelmRelease, targetSource *helmv2beta1.HelmRelease) error {
	currentResource.Spec = targetSource.Spec
	if err := a.Client.Update(ctx, currentResource); err != nil {
		return err
	}
	return nil
}

// findSourceKind determines the type of the application's source.
// An application can only have one specified source type. In case of none or multiple source types are specified,
// this function returns an error. If successful, it returns the string representation of the source type.
func findSourceKind(app *applicationapi.Application) (string, error) {
	validCount := 0
	validKind := ""

	// Check for each type of source in the application and count the number of valid sources
	gitRepo := app.Spec.Source.GitRepo
	if gitRepo != nil {
		validCount++
		validKind = GitRepoKind
	}
	helmRepo := app.Spec.Source.HelmRepo
	if helmRepo != nil {
		validCount++
		validKind = HelmRepoKind
	}
	ociRepo := app.Spec.Source.OCIRepo
	if ociRepo != nil {
		validCount++
		validKind = OCIRepoKind
	}

	// If more than one or no sources are valid, return an error
	if validCount != 1 {
		return "", fmt.Errorf("only one type of source can be specified. The current specified count is %d", validCount)
	}

	// If only one source is valid, return its type
	return validKind, nil
}

// generateKustomizationName constructs a unique name for Kustomization based on the provided application,
// synchronization policy, cluster kind and cluster name. The resulting name is formatted to be lower-case,
// and is truncated to a maximum of 63 characters if needed.

// generateKustomizationName constructs a unique name for Kustomization based on the provided application,
func generateKustomizationName(app *applicationapi.Application, syncPolicy *applicationapi.ApplicationSyncPolicy, clusterKind, clusterName string) string {
	name := app.Name + "-" + syncPolicy.Name + "-" + clusterKind + "-" + clusterName
	name = strings.ToLower(name)
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}

// generateHelmReleaseName constructs a unique name for HelmRelease based on the provided application,
func generateHelmReleaseName(app *applicationapi.Application, syncPolicy *applicationapi.ApplicationSyncPolicy, clusterKind, clusterName string) string {
	name := app.Name + "-" + syncPolicy.Name + "-" + clusterKind + "-" + clusterName
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
