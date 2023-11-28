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
	"time"

	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	fleetv1a1 "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	"kurator.dev/kurator/pkg/fleet-manager/plugin"
	"kurator.dev/kurator/pkg/infra/util"
)

func (f *FleetManager) reconcileKyvernoPlugin(ctx context.Context, fleet *fleetv1a1.Fleet, fleetClusters map[ClusterKey]*fleetCluster) (kube.ResourceList, ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if fleet.Spec.Plugin.Policy == nil ||
		fleet.Spec.Plugin.Policy.Kyverno == nil {
		// reconcilePluginResources will delete all resources if plugin is nil
		return nil, ctrl.Result{}, nil
	}

	kyvernoCfg := fleet.Spec.Plugin.Policy.Kyverno

	fleetNN := types.NamespacedName{
		Namespace: fleet.Namespace,
		Name:      fleet.Name,
	}

	fleetOwnerRef := ownerReference(fleet)
	var resources kube.ResourceList
	for key, cluster := range fleetClusters {
		b, err := plugin.RenderKyverno(f.Manifests, fleetNN, fleetOwnerRef, plugin.KubeConfigSecretRef{
			Name:       key.Name,
			SecretName: cluster.Secret,
			SecretKey:  cluster.SecretKey,
		}, fleet.Spec.Plugin.Policy.Kyverno)
		if err != nil {
			return nil, ctrl.Result{}, err
		}

		// apply kyverno resources
		kyvernoResources, err := util.PatchResources(b)
		if err != nil {
			return nil, ctrl.Result{}, err
		}
		resources = append(resources, kyvernoResources...)
	}

	log.V(4).Info("wait for kyverno helm release to be reconciled")
	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	// After CRDs are created, start to install pod security policy
	if kyvernoCfg.PodSecurity != nil {
		for key, cluster := range fleetClusters {
			// generate policies for pod security admission
			b, err := plugin.RenderKyvernoPolicy(f.Manifests, fleetNN, fleetOwnerRef, plugin.KubeConfigSecretRef{
				Name:       key.Name,
				SecretName: cluster.Secret,
				SecretKey:  cluster.SecretKey,
			}, kyvernoCfg)
			if err != nil {
				return nil, ctrl.Result{}, err
			}

			kyvernoPolicyResources, err := util.PatchResources(b)
			if err != nil {
				return nil, ctrl.Result{}, err
			}
			resources = append(resources, kyvernoPolicyResources...)
		}
	}

	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	return resources, ctrl.Result{}, nil
}
