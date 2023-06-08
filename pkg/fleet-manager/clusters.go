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
	"time"

	karmadaclientset "github.com/karmada-io/karmada/pkg/generated/clientset/versioned"
	"github.com/karmada-io/karmada/pkg/karmadactl/join"
	"github.com/karmada-io/karmada/pkg/karmadactl/options"
	"github.com/karmada-io/karmada/pkg/karmadactl/unjoin"
	"github.com/karmada-io/karmada/pkg/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	kubeclient "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

// TODO: rename to FleetCluster?
type ClusterInterface interface {
	IsReady() bool
	GetObject() client.Object
	GetSecretName() string
	GetSecretKey() string
}

const (
	ClusterKind         = "Cluster"
	AttachedClusterKind = "AttachedCluster"
)

func (f *FleetManager) reconcileClusters(ctx context.Context, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	controlplane := fleet.Annotations[fleetapi.ControlplaneAnnotation]
	controlplaneSpecified := true
	if len(controlplane) == 0 {
		controlplaneSpecified = false
	}

	fleetKey := client.ObjectKeyFromObject(fleet)
	log := ctrl.LoggerFrom(ctx).WithValues("fleet", fleetKey)
	var unreadyClusters int32
	var result ctrl.Result
	var readyClusters []ClusterInterface
	clusterMap := make(map[string]struct{}, len(fleet.Spec.Clusters))
	// Loop over cluster, and add labels to the cluster
	for _, cluster := range fleet.Spec.Clusters {
		// cluster namespace can be not set, always use fleet namespace as a fleet can only include clusters in the same namespace.
		clusterKey := types.NamespacedName{Name: cluster.Name, Namespace: fleet.Namespace}
		currentCluster, err := f.getFleetClusterInterface(ctx, cluster.Kind, clusterKey)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "unable to fetch cluster", "cluster", clusterKey, "kind", cluster.Kind)
				return result, err
			}
			unreadyClusters++
			continue
		}

		// In case multiple clusters of different kinds have the same name.
		clusterMap[generateClusterNameInKarmada(currentCluster)] = struct{}{}

		// label the cluster
		if currentCluster.GetObject().GetLabels() == nil {
			currentCluster.GetObject().SetLabels(make(map[string]string))
		}
		if currentCluster.GetObject().GetLabels()[FleetLabel] != fleet.Name {
			currentCluster.GetObject().GetLabels()[FleetLabel] = fleet.Name
			err = f.Update(ctx, currentCluster.GetObject())
			if err != nil {
				log.Error(err, "unable to label cluster", "cluster", clusterKey)
				return ctrl.Result{}, err
			}
		}
		// Register the ready cluster to the control plane
		if currentCluster.IsReady() {
			readyClusters = append(readyClusters, currentCluster)
		} else {
			unreadyClusters++
		}
	}

	fleet.Status.ReadyClusters = int32(len(readyClusters))
	fleet.Status.UnReadyClusters = unreadyClusters

	var controlplaneRestConfig *restclient.Config
	if controlplaneSpecified {
		var kubeconfig corev1.Secret
		controlPlaneSecretKey := types.NamespacedName{Name: "kubeconfig", Namespace: fleet.Namespace}
		err := f.Get(ctx, controlPlaneSecretKey, &kubeconfig)
		if err != nil {
			return result, err
		}

		controlplaneRestConfig, err = clientcmd.RESTConfigFromKubeConfig(kubeconfig.Data["kubeconfig"])
		if err != nil {
			log.Error(err, "build restconfig for controlplane failed")
			return result, fmt.Errorf("build restconfig for controlplane failed %v", err)
		}
		for _, cluster := range readyClusters {
			err := f.joinCluster(ctx, controlplaneRestConfig, cluster)
			if err != nil {
				log.Error(err, "Join cluster failed")
				return result, err
			}
		}
	}

	// Handle cluster unjoin
	var clusterList clusterv1alpha1.ClusterList
	err := f.Client.List(ctx, &clusterList,
		client.InNamespace(fleet.Namespace),
		client.MatchingLabels{FleetLabel: fleet.Name})
	if err != nil {
		return result, err
	}

	var attachedClusterList clusterv1alpha1.AttachedClusterList
	err = f.Client.List(ctx, &attachedClusterList,
		client.InNamespace(fleet.Namespace),
		client.MatchingLabels{FleetLabel: fleet.Name})
	if err != nil {
		return result, err
	}

	var labeledCluster []ClusterInterface

	for _, cluster := range clusterList.Items {
		tmpCluster := cluster
		labeledCluster = append(labeledCluster, &tmpCluster)
	}

	for _, attachedCluster := range attachedClusterList.Items {
		tmpAttachedCluster := attachedCluster
		labeledCluster = append(labeledCluster, &tmpAttachedCluster)
	}

	for _, cluster := range labeledCluster {
		if _, ok := clusterMap[generateClusterNameInKarmada(cluster)]; !ok {
			if controlplaneSpecified {
				err = f.unjoinCluster(ctx, controlplaneRestConfig, cluster)
				if err != nil {
					log.Error(err, "Unjoin cluster failed")
					return result, err
				}
			}

			// remove label after unjoined
			if cluster.GetObject().GetLabels()[FleetLabel] == fleet.Name {
				delete(cluster.GetObject().GetLabels(), FleetLabel)
				err = f.Update(ctx, cluster.GetObject())
				if err != nil {
					log.Error(err, "unable to remove cluster label", "cluster", cluster.GetObject().GetName())
					return result, err
				}
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
		currentCluster, err := f.getFleetClusterInterface(ctx, cluster.Kind, clusterKey)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(err, "unable to fetch cluster", "cluster", clusterKey)
				return result, err
			}
			log.Info("cluster not found maybe deleted or not created", "cluster", clusterKey)
		} else {
			if currentCluster.GetObject().GetLabels()[FleetLabel] == fleet.Name {
				delete(currentCluster.GetObject().GetLabels(), FleetLabel)
				err = f.Update(ctx, currentCluster.GetObject())
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
		ClusterName:      generateClusterNameInKarmada(cluster),
	}

	// check if already joined.
	alreadyJoined, err := isClusterAlreadyJoined(controlPlane, clusterRestConfig)

	if err != nil {
		return err
	}
	// if already joined, return directly.
	if alreadyJoined {
		return nil
	}

	err = option.RunJoinCluster(controlPlane, clusterRestConfig)
	if err != nil {
		return fmt.Errorf("join cluster %v failed %v", client.ObjectKeyFromObject(cluster.GetObject()), err)
	}
	return nil
}

// isClusterAlreadyJoined check if current cluster is already joined.
func isClusterAlreadyJoined(controlPlaneRestConfig, clusterConfig *restclient.Config) (bool, error) {
	karmadaClient := karmadaclientset.NewForConfigOrDie(controlPlaneRestConfig)
	clusterKubeClient := kubeclient.NewForConfigOrDie(clusterConfig)
	id, err := util.ObtainClusterID(clusterKubeClient)
	if err != nil {
		return false, err
	}

	ok, _, err := util.IsClusterIdentifyUnique(karmadaClient, id)
	if err != nil {
		return false, err
	}

	if !ok {
		return true, nil
	}
	return false, nil
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
		ClusterName:      generateClusterNameInKarmada(cluster),
		Wait:             60 * time.Second,
	}
	err = option.RunUnJoinCluster(controlPlane, clusterRestConfig)
	if err != nil {
		return fmt.Errorf("unjoin cluster %v failed %v", client.ObjectKeyFromObject(cluster.GetObject()), err)
	}
	return nil
}

// TODO: rewrite it to reduce conflict
// generateClusterNameInKarmada generate the name for karmada
func generateClusterNameInKarmada(cluster ClusterInterface) string {
	// to ensure a unique name in Karmada, add the suffix of the kind to avoid the possibility of different kind clusters having the same name.
	name := cluster.GetObject().GetName() + "-" + strings.ToLower(cluster.GetObject().GetObjectKind().GroupVersionKind().Kind)
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}
