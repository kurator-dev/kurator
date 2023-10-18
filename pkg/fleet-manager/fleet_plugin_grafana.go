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
	"time"

	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	"kurator.dev/kurator/pkg/fleet-manager/plugin"
	"kurator.dev/kurator/pkg/infra/util"
)

// reconcileGrafanaPlugin reconciles the Grafana plugin.
// The fleetClusters parameter is currently unused, but is included to match the function signature of other functions in reconcilePlugins.
func (f *FleetManager) reconcileGrafanaPlugin(ctx context.Context, fleet *fleetapi.Fleet, fleetClusters map[ClusterKey]*fleetCluster) (kube.ResourceList, ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if fleet.Spec.Plugin.Grafana == nil {
		// reconcilePluginResources will delete all resources if plugin is nil
		return nil, ctrl.Result{}, nil
	}

	fleetNN := types.NamespacedName{
		Namespace: fleet.Namespace,
		Name:      fleet.Name,
	}

	fleetOwnerRef := ownerReference(fleet)

	dataSources := make([]*plugin.GrafanaDataSource, 0, 1)
	if fleet.Spec.Plugin.Metric != nil {
		dataSources = append(dataSources, &plugin.GrafanaDataSource{
			Name:       "Thanos",
			SourceType: "prometheus",
			URL:        fmt.Sprintf("http://%s-thanos-query:9090", fleet.Namespace), // HelmRelease will put namespace in the name
			Access:     "proxy",
			IsDefault:  true,
		})
	}

	b, err := plugin.RenderGrafana(f.Manifests, fleetNN, fleetOwnerRef, fleet.Spec.Plugin.Grafana, dataSources)
	if err != nil {
		return nil, ctrl.Result{}, err
	}

	resources, err := util.PatchResources(b)
	if err != nil {
		return nil, ctrl.Result{}, err
	}

	log.V(4).Info("wait for grafana helm release to be reconciled")
	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	return resources, ctrl.Result{}, nil
}
