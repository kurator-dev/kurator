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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=customclusters,shortName=cc
// +kubebuilder:subresource:status

// CustomCluster is the schema for existing node based Kubernetes Cluster API.
type CustomCluster struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the kurator cluster.
	Spec CustomClusterSpec `json:"spec,omitempty"`

	// Current status of the kurator cluster
	Status CustomClusterStatus `json:"status,omitempty"`
}

// CustomClusterSpec defines the desired state of a kurator cluster.
type CustomClusterSpec struct {
	// TODO: any UCS specific configurations that does not exist in upstream cluster api

	// MachineRef is the reference of nodes for provisioning a kurator cluster.
	// +optional
	MachineRef *corev1.ObjectReference `json:"machineRef,omitempty"`

	// CNIConfig is the configuration for the CNI of the cluster on VMs.
	CNI CNIConfig `json:"cni"`
}

type CNIConfig struct {
	// Type is the type of CNI. The default value is calico and can be ["calico", "cilium", "canal", "flannel"]
	Type string `json:"type"`
}

type CustomClusterPhase string

const (
	// RunningPhase represents the cluster on VMs is creating. In this phase, the worker named ends in "init" is running to initialize the cluster on VMs
	RunningPhase CustomClusterPhase = "Running"

	// SucceededPhase represents the cluster on VMs is created successful. In this phase, the worker named ends in "init" is completed
	SucceededPhase CustomClusterPhase = "Succeeded"

	// TerminatingPhase represents the cluster on VMs is being cleared. In this phase, the worker named ends in "terminate" is running to clear the cluster on VMs
	TerminatingPhase CustomClusterPhase = "Terminating"

	// InitFailedPhase represents something is wrong when creating the cluster on VMs. In this phase, the worker named ends in "init" is in error
	InitFailedPhase CustomClusterPhase = "InitFailed"

	// TerminateFailedPhase represents something is wrong when clearing the cluster on VMs. In this phase, the worker named ends in "terminate" is in error
	TerminateFailedPhase CustomClusterPhase = "TerminateFailed"
)

// CustomClusterStatus represents the current status of the cluster.
type CustomClusterStatus struct {

	// Phase represents the current phase of customCluster actuation.
	// E.g.  Running, Succeed, Terminating, Failed etc.
	// +optional
	Phase CustomClusterPhase `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CustomClusterList contains a list of CustomCluster.
type CustomClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomCluster `json:"items"`
}
