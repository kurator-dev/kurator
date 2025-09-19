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

package fleet

import (
	"context"
	"time"

	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	"kurator.dev/kurator/pkg/fleet-manager/plugin"
	"kurator.dev/kurator/pkg/infra/util"
)

// reconcileFlaggerPlugin reconciles the Flagger plugin.
// The fleetClusters parameter is currently unused, but is included to match the function signature of other functions in reconcilePlugins.
func (f *FleetManager) reconcileFlaggerPlugin(ctx context.Context, fleet *fleetapi.Fleet, fleetClusters map[ClusterKey]*FleetCluster) (kube.ResourceList, ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	flaggerCfg := fleet.Spec.Plugin.Flagger

	if flaggerCfg == nil {
		// reconcilePluginResources will delete all resources if plugin is nil
		return nil, ctrl.Result{}, nil
	}

	fleetNN := types.NamespacedName{
		Namespace: fleet.Namespace,
		Name:      fleet.Name,
	}

	fleetOwnerRef := ownerReference(fleet)
	var resources kube.ResourceList

	for key, cluster := range fleetClusters {
		b, err := plugin.RenderFlagger(f.Manifests, fleetNN, fleetOwnerRef, plugin.KubeConfigSecretRef{
			Name:       key.Name,
			SecretName: cluster.Secret,
			SecretKey:  cluster.SecretKey,
		}, flaggerCfg)
		if err != nil {
			return nil, ctrl.Result{}, err
		}

		// apply flagger helm resources
		flaggerResources, err := util.PatchResources(b)
		if err != nil {
			return nil, ctrl.Result{}, err
		}
		resources = append(resources, flaggerResources...)

		// install public testloader if needed
		if flaggerCfg.PublicTestloader {
			b, err := plugin.RenderRolloutTestloader(f.Manifests, fleetNN, fleetOwnerRef, plugin.KubeConfigSecretRef{
				Name:       key.Name,
				SecretName: cluster.Secret,
				SecretKey:  cluster.SecretKey,
			}, flaggerCfg)
			if err != nil {
				return nil, ctrl.Result{}, err
			}

			// apply flagger helm resources
			testloaderResources, err := util.PatchResources(b)
			if err != nil {
				return nil, ctrl.Result{}, err
			}
			resources = append(resources, testloaderResources...)
		}
	}

	log.V(4).Info("wait for flagger helm release to be reconciled")
	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	return resources, ctrl.Result{}, nil
}
