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
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"helm.sh/helm/v3/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	"kurator.dev/kurator/pkg/fleet-manager/plugin"
	"kurator.dev/kurator/pkg/infra/util"
)

var BROKER_NS string = "submariner-k8s-broker"

func getBrokerConfig(ctx context.Context, key ClusterKey, cluster *FleetCluster) (map[string]interface{}, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	st_name := fmt.Sprintf("%s-%s-%s-client-token", BROKER_NS, plugin.SubMarinerBrokerComponentName, key.Name)
	sts, err := cluster.Client.KubeClient().CoreV1().Secrets(BROKER_NS).Get(ctx, st_name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	broker_ca := base64.StdEncoding.EncodeToString(sts.Data["ca.crt"])
	broker_token := string(sts.Data["token"])

	endpoints, err := cluster.Client.KubeClient().CoreV1().Endpoints("default").Get(context.TODO(), "kubernetes", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	broker_url := ""
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			for _, port := range subset.Ports {
				if port.Name == "https" {
					broker_url = fmt.Sprintf("%s:%d\n", addr.IP, port.Port)
					break
				}
			}
		}
	}
	if broker_url == "" {
		return nil, errors.New("broker url not found")
	}

	broker_cfg := map[string]interface{}{
		"ca":     broker_ca,
		"token":  broker_token,
		"server": broker_url,
	}
	return broker_cfg, nil
}

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

	// At least two member required
	if len(fleetClusters) < 2 {
		return nil, ctrl.Result{}, errors.New("fleetClusters number < 2")
	}

	brokerClusterKey := ClusterKey{
		Kind: fleet.Spec.Clusters[0].Kind,
		Name: fleet.Spec.Clusters[0].Name,
	}

	// Install broker in the first member cluster
	log.V(0).Info("broker will be installed in " + brokerClusterKey.Name)
	brokerCluster := fleetClusters[brokerClusterKey]
	b, err := plugin.RenderSubmarinerBroker(f.Manifests, fleetNN, fleetOwnerRef, plugin.KubeConfigSecretRef{
		Name:       brokerClusterKey.Name,
		SecretName: brokerCluster.Secret,
		SecretKey:  brokerCluster.SecretKey,
	}, submarinerCfg)
	if err != nil {
		return nil, ctrl.Result{}, err
	}

	brokerResources, err := util.PatchResources(b)
	if err != nil {
		return nil, ctrl.Result{}, err
	}
	resources = append(resources, brokerResources...)

	log.V(0).Info("wait for submariner broker helm release to be reconciled")
	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	broker_cfg, err := getBrokerConfig(ctx, brokerClusterKey, brokerCluster)
	if err != nil {
		log.V(0).Error(err, "failed to get broker info")
		return nil, ctrl.Result{}, err
	}

	// Install operator in all member clusters
	for key, cluster := range fleetClusters {
		b, err := plugin.RenderSubmarinerOperator(f.Manifests, fleetNN, fleetOwnerRef, plugin.KubeConfigSecretRef{
			Name:       key.Name,
			SecretName: cluster.Secret,
			SecretKey:  cluster.SecretKey,
		}, submarinerCfg, broker_cfg)
		if err != nil {
			log.V(0).Error(err, "failed to render submariner operator")
			return nil, ctrl.Result{}, err
		}

		operatorResources, err := util.PatchResources(b)
		if err != nil {
			log.V(0).Error(err, "failed to render submariner operator")
			return nil, ctrl.Result{}, err
		}
		resources = append(resources, operatorResources...)
	}

	log.V(0).Info("wait for submariner operator helm release to be reconciled")
	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}
	log.V(0).Info("Submariner helm release is ready!")
	return resources, ctrl.Result{}, nil
}
