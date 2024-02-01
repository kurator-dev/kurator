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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fleetv1a1 "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	kurator "kurator.dev/kurator/pkg/client-go/generated/clientset/versioned"
)

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
func CreateFleet(client kurator.Interface, fleet *fleetv1a1.Fleet) error {
	_, err := client.FleetV1alpha1().Fleets(fleet.Namespace).Create(context.TODO(), fleet, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return UpdateFleet(client, fleet)
		} else {
			return err
		}
	}
	return nil
}

// UpdateAttachedCluster update AttachedCluster
func UpdateFleet(client kurator.Interface, fleet *fleetv1a1.Fleet) error {
	_, err := client.FleetV1alpha1().Fleets(fleet.Namespace).Update(context.TODO(), fleet, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
