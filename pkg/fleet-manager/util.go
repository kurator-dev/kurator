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
	"istio.io/istio/pkg/util/sets"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	fleetv1a1 "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

const (
	FleetNameLabel  = "fleet.kurator.dev/name"
	FleetPluginName = "fleet.kurator.dev/plugin"

	ManagedByLabel        = "app.kubernetes.io/managed-by"
	ManagedByFleetManager = "fleet-manager"
)

func fleetResourceLables(fleetName string) client.MatchingLabels {
	return map[string]string{
		ManagedByLabel: "fleet-manager",
		FleetNameLabel: fleetName,
	}
}

func fleetMetricResourceLables(fleetName string) client.MatchingLabels {
	return map[string]string{
		ManagedByLabel:  "fleet-manager",
		FleetNameLabel:  fleetName,
		FleetPluginName: "metric",
	}
}

func isReady(conditions []metav1.Condition) bool {
	for _, c := range conditions {
		if c.Type == "Ready" && c.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func convertToSubset(endpoints sets.Set[string]) []corev1.EndpointSubset {
	address := make([]corev1.EndpointAddress, 0, len(endpoints))
	for _, ip := range sets.SortedList(endpoints) {
		address = append(address, corev1.EndpointAddress{
			IP: ip,
		})
	}
	return []corev1.EndpointSubset{
		{
			Addresses: address,
			Ports: []corev1.EndpointPort{
				{
					Name:        "grpc",
					Port:        10901,
					Protocol:    corev1.ProtocolTCP,
					AppProtocol: pointer.String("grpc"),
				},
			},
		},
	}
}

func ownerReference(fleet *fleetv1a1.Fleet) *metav1.OwnerReference {
	return &metav1.OwnerReference{
		APIVersion: fleetv1a1.GroupVersion.String(),
		Kind:       "Fleet", // TODO: use pkg typemeta
		Name:       fleet.Name,
		UID:        fleet.UID,
	}
}
