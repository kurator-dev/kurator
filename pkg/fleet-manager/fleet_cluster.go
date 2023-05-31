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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"

	clusterv1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	kclient "kurator.dev/kurator/pkg/client"
)

type fleetCluster struct {
	Secret    string
	SecretKey string
	client    *kclient.Client
}

type ClusterKey struct {
	Kind string
	Name string
}

func (f *FleetManager) buildFleetClusters(ctx context.Context, fleet *fleetapi.Fleet) (map[ClusterKey]*fleetCluster, error) {
	log := ctrl.LoggerFrom(ctx)

	res := make(map[ClusterKey]*fleetCluster, len(fleet.Spec.Clusters))
	for _, c := range fleet.Spec.Clusters {
		clusterKey := types.NamespacedName{Namespace: fleet.Namespace, Name: c.Name}
		clusterInterface, err := f.getFleetClusterInterface(ctx, c.Kind, clusterKey)
		// TODO: should we make it work
		if err != nil {
			return nil, err
		}

		if !clusterInterface.IsReady() {
			log.V(4).Info("cluster is not ready", "cluster", clusterKey)
			continue
		}

		kclient, err := f.clientForCluster(fleet.Namespace, clusterInterface)
		if err != nil {
			return nil, err
		}
		res[ClusterKey{Kind: c.Kind, Name: c.Name}] = &fleetCluster{
			Secret:    clusterInterface.GetSecretName(),
			SecretKey: clusterInterface.GetSecretKey(),
			client:    kclient,
		}
	}

	return res, nil
}

func (f *FleetManager) getFleetClusterInterface(ctx context.Context, kind string, nn types.NamespacedName) (ClusterInterface, error) {
	switch kind {
	case ClusterKind, "":
		cluster := &clusterv1alpha1.Cluster{}
		if err := f.Get(ctx, nn, cluster); err != nil {
			return nil, err
		}
		return cluster, nil
	case AttachedClusterKind:
		attachedCluster := &clusterv1alpha1.AttachedCluster{}
		if err := f.Get(ctx, nn, attachedCluster); err != nil {
			return nil, err
		}
		return attachedCluster, nil
	default:
		return nil, fmt.Errorf("unsupported cluster kind")
	}
}

func (f *FleetManager) clientForCluster(ns string, cluster ClusterInterface) (*kclient.Client, error) {
	secret := &corev1.Secret{}
	nn := types.NamespacedName{Namespace: ns, Name: cluster.GetSecretName()}
	if err := f.Client.Get(context.Background(), nn, secret); err != nil {
		return nil, err
	}

	kubeconfig, ok := secret.Data[cluster.GetSecretKey()]
	if !ok {
		return nil, fmt.Errorf("key %q not found in secret %s/%s", cluster.GetSecretKey(), secret.Namespace, secret.Name)
	}

	rest, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return kclient.NewClient(kclient.NewRESTClientGetter(rest))
}
