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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FleetPhase string

const (
	// PendingPhase is the phase when the fleet is not processed.
	PendingPhase FleetPhase = "Pending"
	// RunningPhase is the phase when fleet control plane is being installed.
	RunningPhase FleetPhase = "Running"
	// FailedPhase is the phase when fleet control plane installation installed.
	FailedPhase FleetPhase = "Failed"
	// ReadyPhase is the phase when fleet control plane installation finished successfully.
	ReadyPhase FleetPhase = "Ready"
	// TerminatingPhase is the phase when fleet control plane is terminating.
	TerminatingPhase FleetPhase = "Terminating"
	// TerminateFailedPhase is the phase when fleet control plane terminate failed.
	TerminateFailedPhase FleetPhase = "TerminateFailed"
	// TerminateSucceededPhase is the phase when fleet control plane is terminated successfully.
	TerminateSucceededPhase FleetPhase = "TerminateSucceeded"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,categories=kurator-dev
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase of the fleet"

// Fleet represents a group of clusters, it is to consistently manage a group of clusters.
type Fleet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FleetSpec   `json:"spec,omitempty"`
	Status            FleetStatus `json:"status,omitempty"`
}

// FleetSpec defines the desired state of the fleet
type FleetSpec struct {
	// Clusters represents the clusters that would be registered to the fleet.
	// Note: only kurator cluster is supported now
	// TODO: add attached cluster support?
	Clusters []*corev1.ObjectReference `json:"clusters,omitempty"`

	// TODO: support cluster selector?
	// TODO: add options to allow customize fleet control plane if neccessary. And in future this could not be karmada.
}

// FleetStatus defines the observed state of the fleet
type FleetStatus struct {
	// CredentialSecret is the secret name that holds credentials used for accessing the fleet control plane.
	CredentialSecret string `json:"credentialSecret,omitempty"`

	// Phase represents the current phase of fleet.
	// E.g. Pending, Running, Terminating, Failed, Ready, etc.
	// +optional
	Phase FleetPhase `json:"phase,omitempty"`

	// TODO: add conditions fields if needed

	// A brief CamelCase message indicating details about why the fleet is in this state.
	// +optional
	Reason string `json:"reason,omitempty"`

	// TODO: healthy/unhealthy members cluster
	// Total number of ready clusters, ready to deploy .
	ReadyClusters int32 `json:"readyClusters,omitempty"`

	// Total number of unready clusters, not ready for use.
	UnReadyClusters int32 `json:"unReadyClusters,omitempty"`
}

// FleetList contains a list of fleets.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FleetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Fleet `json:"items"`
}
