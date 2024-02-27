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
	"k8s.io/apimachinery/pkg/types"

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

// CreateAttachedCluster create or update AttachedCluster.
func CreateOrUpdateAttachedCluster(client kurator.Interface, attachedCluster *clusterv1a1.AttachedCluster) error {
	_, createErr := client.ClusterV1alpha1().AttachedClusters(attachedCluster.GetNamespace()).Create(context.TODO(), attachedCluster, metav1.CreateOptions{})
	if createErr != nil {
		if apierrors.IsAlreadyExists(createErr) {
			originalAttachedCluster, getErr := client.ClusterV1alpha1().AttachedClusters(attachedCluster.GetNamespace()).Get(context.TODO(), attachedCluster.GetName(), metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}
			modifiedObjectMeta := ModifiedObjectMeta(originalAttachedCluster.ObjectMeta, attachedCluster.ObjectMeta)
			oldAttachedCluster := clusterv1a1.AttachedCluster{
				ObjectMeta: originalAttachedCluster.ObjectMeta,
				Spec:       originalAttachedCluster.Spec,
			}
			modAttachedCluster := clusterv1a1.AttachedCluster{
				ObjectMeta: modifiedObjectMeta,
				Spec:       attachedCluster.Spec,
			}
			attachedClusterPatchData, createPatchErr := CreatePatchData(oldAttachedCluster, modAttachedCluster)
			if createPatchErr != nil {
				return createPatchErr
			}
			_, patchErr := client.ClusterV1alpha1().AttachedClusters(attachedCluster.GetNamespace()).Patch(context.TODO(), attachedCluster.GetName(), types.MergePatchType, attachedClusterPatchData, metav1.PatchOptions{})
			if patchErr != nil {
				return patchErr
			}
		} else {
			return createErr
		}
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
