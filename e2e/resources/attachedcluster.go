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

package resources

import (
	"context"

	"github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1a1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	kurator "kurator.dev/kurator/pkg/client-go/generated/clientset/versioned"
)

func NewAttachedCluster(namespace string, name string, config clusterv1a1.SecretKeyRef) *clusterv1a1.AttachedCluster {
	return &clusterv1a1.AttachedCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "cluster.kurator.dev/v1alpha1",
			Kind:       "AttachedCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: clusterv1a1.AttachedClusterSpec{
			Kubeconfig: config,
		},
	}
}

// CreateAttachedCluster create AttachedCluster.
func CreateAttachedCluster(client kurator.Interface, attachedCluster *clusterv1a1.AttachedCluster) error {
	_, err := client.ClusterV1alpha1().AttachedClusters(attachedCluster.Namespace).Create(context.TODO(), attachedCluster, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return UpdateAttachedCluster(client, attachedCluster)
		} else {
			return err
		}
	}
	return nil
}

// UpdateAttachedCluster update AttachedCluster
func UpdateAttachedCluster(client kurator.Interface, attachedCluster *clusterv1a1.AttachedCluster) error {
	attachedClusterPresentOnCluster, attacattachedClusterGetErr := client.ClusterV1alpha1().AttachedClusters(attachedCluster.Namespace).Get(context.TODO(), attachedCluster.Name, metav1.GetOptions{})
	if attacattachedClusterGetErr != nil {
		return attacattachedClusterGetErr
	}
	DCattachedcluster := attachedClusterPresentOnCluster.DeepCopy()
	DCattachedcluster.Spec = attachedCluster.Spec
	_, err := client.ClusterV1alpha1().AttachedClusters(DCattachedcluster.Namespace).Update(context.TODO(), DCattachedcluster, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func RemoveAttachedCluster(client kurator.Interface, namespace, name string) error {
	err := client.ClusterV1alpha1().AttachedClusters(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}
	return nil
}

// WaitAttachedClusterFitWith wait attachedCluster sync with fit func.
func WaitAttachedClusterFitWith(client kurator.Interface, namespace, name string, fit func(attachedCluster *clusterv1a1.AttachedCluster) bool) {
	gomega.Eventually(func() bool {
		attachedClusterPresentOnCluster, err := client.ClusterV1alpha1().AttachedClusters(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false
		}
		return fit(attachedClusterPresentOnCluster)
	}, pollTimeoutInHostCluster, pollIntervalInHostCluster).Should(gomega.Equal(true))
}
