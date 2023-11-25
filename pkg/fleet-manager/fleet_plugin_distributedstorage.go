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

	"kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	"kurator.dev/kurator/pkg/fleet-manager/plugin"
	"kurator.dev/kurator/pkg/infra/util"
)

func (f *FleetManager) reconcileDistributedStoragePlugin(ctx context.Context, fleet *v1alpha1.Fleet, fleetClusters map[ClusterKey]*fleetCluster) (kube.ResourceList, ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	distributedStorageCfg := fleet.Spec.Plugin.DistributedStorage

	if distributedStorageCfg == nil {
		// reconcilePluginResources will delete all resources if plugin is nil
		return nil, ctrl.Result{}, nil
	}

	fleetNN := types.NamespacedName{
		Namespace: fleet.Namespace,
		Name:      fleet.Name,
	}

	fleetOwnerRef := ownerReference(fleet)
	var resources kube.ResourceList

	// First install rook-operator for the specified multicluster.
	for key, cluster := range fleetClusters {
		b, err := plugin.RendeStorageOperator(f.Manifests, fleetNN, fleetOwnerRef, plugin.RenderableFleetCluster{
			Name:       key.Name,
			SecretName: cluster.Secret,
			SecretKey:  cluster.SecretKey,
		}, distributedStorageCfg)
		if err != nil {
			return nil, ctrl.Result{}, err
		}

		// apply Rook helm resources
		rookResources, err := util.PatchResources(b)
		if err != nil {
			return nil, ctrl.Result{}, err
		}
		resources = append(resources, rookResources...)
	}

	log.V(4).Info("wait for rook helm release to be reconciled")
	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	// After Rook operator are created, starts to install rook-ceph
	if distributedStorageCfg.Storage != nil {
		for key, cluster := range fleetClusters {
			b, err := plugin.RenderClusterStorage(f.Manifests, fleetNN, fleetOwnerRef, plugin.RenderableFleetCluster{
				Name:       key.Name,
				SecretName: cluster.Secret,
				SecretKey:  cluster.SecretKey,
			}, distributedStorageCfg)
			if err != nil {
				return nil, ctrl.Result{}, err
			}

			rookCephResources, err := util.PatchResources(b)
			if err != nil {
				return nil, ctrl.Result{}, err
			}
			resources = append(resources, rookCephResources...)
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
