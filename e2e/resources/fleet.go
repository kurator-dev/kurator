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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	fleetv1a1 "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	kurator "kurator.dev/kurator/pkg/client-go/generated/clientset/versioned"
)

// NewFleet will build a Fleet object.
func NewFleet(namespace string, name string, clusters []*corev1.ObjectReference) *fleetv1a1.Fleet {
	return &fleetv1a1.Fleet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "fleet.kurator.dev/v1alpha1",
			Kind:       "Fleet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: fleetv1a1.FleetSpec{
			Clusters: clusters,
		},
	}
}

// CreateAttachedCluster create AttachedCluster.
func CreateOrUpdateFleet(client kurator.Interface, fleet *fleetv1a1.Fleet) error {
	_, createErr := client.FleetV1alpha1().Fleets(fleet.GetNamespace()).Create(context.TODO(), fleet, metav1.CreateOptions{})
	if createErr != nil {
		if apierrors.IsAlreadyExists(createErr) {
			originalFleet, getErr := client.FleetV1alpha1().Fleets(fleet.GetNamespace()).Get(context.TODO(), fleet.GetName(), metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}
			fleet.ResourceVersion = originalFleet.ResourceVersion
			fleetPatchData, createPatchErr := CreatePatchData(originalFleet, fleet)
			if createPatchErr != nil {
				return createPatchErr
			}
			if _, patchErr := client.FleetV1alpha1().Fleets(fleet.GetNamespace()).Patch(context.TODO(), fleet.GetName(), types.MergePatchType, fleetPatchData, metav1.PatchOptions{}); patchErr != nil {
				return patchErr
			}
		} else {
			return createErr
		}
	}
	return nil
}

func RemoveFleet(client kurator.Interface, namespace, name string) error {
	err := client.FleetV1alpha1().Fleets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}
	return nil
}

// WaitAttachedClusterFitWith wait fleet sync with fit func.
func WaitFleetFitWith(client kurator.Interface, namespace, name string, fit func(fleeet *fleetv1a1.Fleet) bool) {
	gomega.Eventually(func() bool {
		fleetPresentOnCluster, err := client.FleetV1alpha1().Fleets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false
		}
		return fit(fleetPresentOnCluster)
	}, pollTimeoutInHostCluster, pollIntervalInHostCluster).Should(gomega.Equal(true))
}
