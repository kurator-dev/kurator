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

// Must match the namespace in pkg/fleet-manager/manifests/plugins/sm-broker.yaml
const brokerNs string = "submariner-k8s-broker"

func getBrokerConfig(ctx context.Context, clusterName string, cluster *FleetCluster) (map[string]interface{}, error) {
	brokerSecret := fmt.Sprintf("%s-%s-%s-client-token", brokerNs, plugin.SubMarinerBrokerComponentName, clusterName)
	secrets, err := cluster.Client.KubeClient().CoreV1().Secrets(brokerNs).Get(ctx, brokerSecret, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	endpoints, err := cluster.Client.KubeClient().CoreV1().Endpoints("default").Get(ctx, "kubernetes", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var brokerUrl string
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			for _, port := range subset.Ports {
				if port.Name == "https" {
					brokerUrl = fmt.Sprintf("%s:%d\n", addr.IP, port.Port)
					break
				}
			}
		}
	}
	if brokerUrl == "" {
		return nil, errors.New("broker url not found")
	}

	brokerCfg := map[string]interface{}{
		"ca":        base64.StdEncoding.EncodeToString(secrets.Data["ca.crt"]),
		"token":     string(secrets.Data["token"]),
		"server":    brokerUrl,
		"namespace": brokerNs,
	}
	return brokerCfg, nil
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

	var brokerClusterName string = submarinerCfg.BrokerCluster
	var brokerCluster *FleetCluster

	if submarinerCfg.BrokerCluster == "" {
		// Install broker in the first member cluster
		brokerClusterName = fleet.Spec.Clusters[0].Name
		log.V(4).Info("broker cluster not specified, using the first member cluster", "brokerClusterName", brokerClusterName)
	}

	for key, cluster := range fleetClusters {
		if key.Name == brokerClusterName {
			brokerCluster = cluster
			b, err := plugin.RenderSubmarinerBroker(f.Manifests, fleetNN, fleetOwnerRef, plugin.KubeConfigSecretRef{
				Name:       brokerClusterName,
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

			log.V(4).Info("broker will be installed in " + brokerClusterName)
			break
		}
	}

	log.V(4).Info("wait for submariner broker helm release to be reconciled")
	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	brokerCfg, err := getBrokerConfig(ctx, brokerClusterName, brokerCluster)
	if err != nil {
		log.V(4).Error(err, "failed to get broker info")
		return nil, ctrl.Result{}, err
	}

	// Install operator in all member clusters
	for key, cluster := range fleetClusters {
		globalcidr, ok := submarinerCfg.Globalcidrs[key.Name]
		if !ok {
			globalcidr = ""
		}
		b, err := plugin.RenderSubmarinerOperator(f.Manifests, fleetNN, fleetOwnerRef, plugin.KubeConfigSecretRef{
			Name:       key.Name,
			SecretName: cluster.Secret,
			SecretKey:  cluster.SecretKey,
		}, submarinerCfg, brokerCfg, globalcidr)
		if err != nil {
			log.V(4).Error(err, "failed to render submariner operator")
			return nil, ctrl.Result{}, err
		}

		operatorResources, err := util.PatchResources(b)
		if err != nil {
			log.V(4).Error(err, "failed to render submariner operator")
			return nil, ctrl.Result{}, err
		}
		resources = append(resources, operatorResources...)
	}

	log.V(4).Info("wait for submariner operator helm release to be reconciled")
	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	log.V(4).Info("Submariner helm release is ready!")
	return resources, ctrl.Result{}, nil
}
