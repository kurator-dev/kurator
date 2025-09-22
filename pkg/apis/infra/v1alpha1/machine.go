/*
Copyright 2022-2025 Kurator Authors.

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
// +kubebuilder:resource:path=custommachines
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready",description="Indicates if the CustomMachine is ready."

// CustomMachine is the schema for kubernetes nodes.
type CustomMachine struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the kurator cluster.
	Spec CustomMachineSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Current status of the machine.
	Status CustomMachineStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// CustomMachineSpec defines kubernetes cluster master and nodes.
type CustomMachineSpec struct {
	Master []Machine `json:"master,omitempty"`
	Nodes  []Machine `json:"node,omitempty"`
}

// CustomMachineStatus represents the current status of the machine.
type CustomMachineStatus struct {
	// Indicate whether the machines are ready.
	Ready *bool `json:"ready,omitempty"`
}

// Machine defines a node.
type Machine struct {
	// HostName is the hostname of the machine.
	HostName string `json:"hostName"`
	// PrivateIP is the private ip address of the machine.
	PrivateIP string `json:"privateIP"`
	// PublicIP specifies the public IP.
	PublicIP string `json:"publicIP"`
	// Region specifies the region where the machine resides.
	// +optional
	Region *string `json:"region,omitempty"`
	// Zone specifies the zone where the machine resides.
	// +optional
	Zone *string `json:"zone,omitempty"`
	// SSHKeyName is the name of the ssh key to attach to the instance. Valid values are empty string (do not use SSH keys), a valid SSH key name, or omitted (use the default SSH key name)
	// +optional
	SSHKey *corev1.ObjectReference `json:"sshKey,omitempty"`
	// AdditionalTags is an optional set of tags to add to an instance.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CustomMachineList contains a list of CustomMachine.
type CustomMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomMachine `json:"items"`
}
