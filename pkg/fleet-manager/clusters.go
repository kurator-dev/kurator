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

	"github.com/karmada-io/karmada/pkg/karmadactl/join"
	"github.com/karmada-io/karmada/pkg/karmadactl/options"
	"github.com/karmada-io/karmada/pkg/karmadactl/unjoin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

type ClusterInterface interface {
	IsReady() bool
	GetObject() client.Object
	GetSecretName() string
	GetSecretKey() string
	SetAccepted(accepted bool)
}

const (
	ClusterKind         = "Cluster"
	AttachedClusterKind = "AttachedCluster"
)

func (f *FleetManager) reconcileClusters(ctx context.Context, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	fleetKey := client.ObjectKeyFromObject(fleet)
	log := ctrl.LoggerFrom(ctx).WithValues("fleet", fleetKey)
	var unreadyClusters int32
	var result ctrl.Result
	var readyClusters []ClusterInterface
	clusterMap := make(map[string]struct{}, len(fleet.Spec.Clusters))
	// Loop over cluster, and add labels to the cluster
	for _, cluster := range fleet.Spec.Clusters {
		clusterMap[cluster.Name] = struct{}{}
		// cluster namespace can be not set, always use fleet namespace as a fleet can only include clusters in the same namespace.
		clusterKey := types.NamespacedName{Name: cluster.Name, Namespace: fleet.Namespace}

		var currentCLuster ClusterInterface
		var err error
		if cluster.Kind == ClusterKind {
			var cluster clusterv1alpha1.Cluster
			err = f.Get(ctx, clusterKey, &cluster)
			currentCLuster = &cluster
		} else if cluster.Kind == AttachedClusterKind {
			var attachedCluster clusterv1alpha1.AttachedCluster
			err = f.Get(ctx, clusterKey, &attachedCluster)
			currentCLuster = &attachedCluster
		} else {
			log.Error(nil, "unsupported cluster kind", "cluster", clusterKey, "kind", cluster.Kind)
			return result, nil
		}

		if err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "unable to fetch cluster", "cluster", clusterKey)
				return result, err
			}
			log.Info("cluster not found yet", "cluster", clusterKey)
			result.RequeueAfter = RequeueAfter
		} else {
			// label the cluster
			if currentCLuster.GetObject().GetLabels() == nil {
				currentCLuster.GetObject().SetLabels(make(map[string]string))
			}
			if currentCLuster.GetObject().GetLabels()[FleetLabel] != fleet.Name {
				labels := currentCLuster.GetObject().GetLabels()
				labels[FleetLabel] = fleet.Name
				currentCLuster.GetObject().SetLabels(labels)
				currentCLuster.SetAccepted(true)
				err = f.Update(ctx, currentCLuster.GetObject())
				if err != nil {
					log.Error(err, "unable to label cluster", "cluster", clusterKey)
					return ctrl.Result{}, err
				}
			}
			// Register the ready cluster to the control plane
			if currentCLuster.IsReady() {
				readyClusters = append(readyClusters, currentCLuster)
			} else {
				unreadyClusters++
			}
		}
	}

	fleet.Status.ReadyClusters = int32(len(readyClusters))
	fleet.Status.UnReadyClusters = unreadyClusters

	var kubeconfig corev1.Secret
	controlPlaneSecretKey := types.NamespacedName{Name: "kubeconfig", Namespace: fleet.Namespace}
	err := f.Get(ctx, controlPlaneSecretKey, &kubeconfig)
	if err != nil {
		return result, err
	}

	controlplaneRestConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig.Data["kubeconfig"])
	if err != nil {
		log.Error(err, "build restconfig for controlplane failed")
		return result, fmt.Errorf("build restconfig for controlplane failed %v", err)
	}
	for _, cluster := range readyClusters {
		// TODO: check if the cluster is already joined
		err := f.joinCluster(ctx, controlplaneRestConfig, cluster)
		if err != nil {
			log.Error(err, "Join cluster failed")
			return result, err
		}
	}

	// Handle cluster unjoin
	var clusterList clusterv1alpha1.ClusterList
	err = f.Client.List(ctx, &clusterList,
		client.InNamespace(fleet.Namespace),
		client.MatchingLabels{FleetLabel: fleet.Name})
	if err != nil {
		return result, err
	}

	for _, cluster := range clusterList.Items {
		if _, ok := clusterMap[cluster.Name]; !ok {
			err := f.unjoinCluster(ctx, controlplaneRestConfig, &cluster)
			if err != nil {
				log.Error(err, "Unjoin cluster failed")
				return result, err
			}
		}
	}

	return result, nil
}

func (f *FleetManager) reconcileClustersOnDelete(ctx context.Context, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("fleet", types.NamespacedName{Name: fleet.Name, Namespace: fleet.Namespace})
	var result ctrl.Result
	// Loop over cluster, and add labels to the cluster
	for _, cluster := range fleet.Spec.Clusters {
		// cluster namespace can be not set, always use fleet namespace as a fleet can only include clusters in the same namespace.
		clusterKey := types.NamespacedName{Name: cluster.Name, Namespace: fleet.Namespace}

		var currentCLuster ClusterInterface
		var err error
		if cluster.Kind == ClusterKind {
			var cluster clusterv1alpha1.Cluster
			err = f.Get(ctx, clusterKey, &cluster)
			currentCLuster = &cluster
		} else if cluster.Kind == AttachedClusterKind {
			var attachedCluster clusterv1alpha1.AttachedCluster
			err = f.Get(ctx, clusterKey, &attachedCluster)
			currentCLuster = &attachedCluster
		} else {
			log.Error(nil, "unsupported cluster kind", "cluster", clusterKey, "kind", cluster.Kind)
			return result, nil
		}

		if err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "unable to fetch cluster", "cluster", clusterKey)
				return result, err
			}
			log.Info("cluster not found maybe deleted or not created", "cluster", clusterKey)
		} else {
			if currentCLuster.GetObject().GetLabels()[FleetLabel] == fleet.Name {
				delete(currentCLuster.GetObject().GetLabels(), FleetLabel)
				currentCLuster.SetAccepted(false)
				err = f.Update(ctx, currentCLuster.GetObject())
				if err != nil {
					log.Error(err, "unable to remove cluster label", "cluster", clusterKey)
					return result, err
				}
			}
		}
	}

	return result, nil
}

func (f *FleetManager) joinCluster(ctx context.Context, controlPlane *restclient.Config, cluster ClusterInterface) error {
	var secret corev1.Secret

	secretKey := types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetObject().GetNamespace()}

	if err := f.Get(ctx, secretKey, &secret); err != nil {
		return fmt.Errorf("get secret %v for cluster %v failed %v", secretKey, client.ObjectKeyFromObject(cluster.GetObject()), err)
	}
	clusterKubeconfig := secret.Data[cluster.GetSecretKey()]
	clusterRestConfig, err := clientcmd.RESTConfigFromKubeConfig(clusterKubeconfig)
	if err != nil {
		return fmt.Errorf("build restconfig for cluster %v failed %v", client.ObjectKeyFromObject(cluster.GetObject()), err)
	}

	option := join.CommandJoinOption{
		ClusterNamespace: options.DefaultKarmadaClusterNamespace,
		// TODO: how to distinguish different kind with same name
		ClusterName: cluster.GetObject().GetName(),
	}
	err = option.RunJoinCluster(controlPlane, clusterRestConfig)
	if err != nil {
		return fmt.Errorf("join cluster %v failed %v", client.ObjectKeyFromObject(cluster.GetObject()), err)
	}
	return nil
}

func (f *FleetManager) unjoinCluster(ctx context.Context, controlPlane *restclient.Config, cluster ClusterInterface) error {
	var secret corev1.Secret
	secretKey := types.NamespacedName{Name: cluster.GetSecretName(), Namespace: cluster.GetObject().GetNamespace()}

	if err := f.Get(ctx, secretKey, &secret); err != nil {
		return fmt.Errorf("get secret %v for cluster %v failed %v", secretKey, client.ObjectKeyFromObject(cluster.GetObject()), err)
	}
	clusterKubeconfig := secret.Data[cluster.GetSecretKey()]
	clusterRestConfig, err := clientcmd.RESTConfigFromKubeConfig(clusterKubeconfig)
	if err != nil {
		return fmt.Errorf("build restconfig for cluster %v failed %v", client.ObjectKeyFromObject(cluster.GetObject()), err)
	}

	option := unjoin.CommandUnjoinOption{
		ClusterNamespace: options.DefaultKarmadaClusterNamespace,
		// TODO: how to distinguish different kind with same name
		ClusterName: cluster.GetObject().GetName(),
		Wait:        60 * time.Second,
	}
	err = option.RunUnJoinCluster(controlPlane, clusterRestConfig)
	if err != nil {
		return fmt.Errorf("unjoin cluster %v failed %v", client.ObjectKeyFromObject(cluster.GetObject()), err)
	}
	return nil
}
