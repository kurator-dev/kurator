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

	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	"kurator.dev/kurator/pkg/fleet-manager/plugin"
	"kurator.dev/kurator/pkg/infra/util"
)

// reconcileSubmarinerPlugin reconciles the Submariner plugin.
// The fleetClusters parameter is currently unused, but is included to match the function signature of other functions in reconcilePlugins.
func (f *FleetManager) reconcileSubmarinerPlugin(ctx context.Context, fleet *fleetapi.Fleet, fleetClusters map[ClusterKey]*FleetCluster) (kube.ResourceList, ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	submarinerCfg := fleet.Spec.Plugin.SubMariner

	if submarinerCfg == nil {
		// reconcilePluginResources will delete all resources if plugin is nil
		return nil, ctrl.Result{}, nil
	}

	fleetNN := types.NamespacedName{
		Namespace: fleet.Namespace,
		Name:      fleet.Name,
	}

	fleetOwnerRef := ownerReference(fleet)
	var resources kube.ResourceList
	var b []byte
	var err error

	exist_broker := false
	for key, cluster := range fleetClusters {
		if !exist_broker {
			b, err = plugin.RenderSubmarinerBroker(f.Manifests, fleetNN, fleetOwnerRef, plugin.KubeConfigSecretRef{
				Name:       key.Name,
				SecretName: cluster.Secret,
				SecretKey:  cluster.SecretKey,
			}, submarinerCfg)
			exist_broker = true
			if err != nil {
				return nil, ctrl.Result{}, err
			}

			submarinerResources, err := util.PatchResources(b)
			if err != nil {
				return nil, ctrl.Result{}, err
			}
			resources = append(resources, submarinerResources...)
		}
		b, err = plugin.RenderSubmarinerOperator(f.Manifests, fleetNN, fleetOwnerRef, plugin.KubeConfigSecretRef{
			Name:       key.Name,
			SecretName: cluster.Secret,
			SecretKey:  cluster.SecretKey,
		}, submarinerCfg)

		if err != nil {
			return nil, ctrl.Result{}, err
		}

		submarinerResources, err := util.PatchResources(b)
		if err != nil {
			return nil, ctrl.Result{}, err
		}
		resources = append(resources, submarinerResources...)
	}

	log.V(4).Info("wait for submariner helm release to be reconciled")
	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	return resources, ctrl.Result{}, nil
}
