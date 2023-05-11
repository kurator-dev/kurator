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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,categories=kurator-dev
// +kubebuilder:subresource:status
// AttachedCluster is the schema for the external cluster that are not created by kurator.
type AttachedCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AttachedClusterSpec   `json:"spec,omitempty"`
	Status            AttachedClusterStatus `json:"status,omitempty"`
}

type AttachedClusterSpec struct {
	// Kubeconfig represents the secret that contains the credential to access this cluster.
	// +optional
	Kubeconfig SecretKeyRef `json:"kubeconfig,omitempty"`
}

// SecretKeyRef holds the reference to a secret key.
type SecretKeyRef struct {
	// Name is the name of the secret.
	Name string `json:"name"`
	// Key is the key of the secret.
	// If no key is specified, the secret's default key is `value`.
	// +kubebuilder:default:="value"
	Key string `json:"key"`
}

type AttachedClusterStatus struct {
	// Accepted indicates whether the cluster is registered to kurator fleet.
	// +optional
	Accepted bool `json:"accepted"`
	// Ready indicates whether the cluster is ready to be registered with Kurator Fleet.
	// +optional
	Ready bool `json:"ready"`
}

// AttachedClusterList contains a list of AttachedCluster.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AttachedClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AttachedCluster `json:"items"`
}

func (ac *AttachedCluster) IsReady() bool {
	return ac.Status.Ready
}

func (ac *AttachedCluster) GetObject() client.Object {
	return ac
}

func (ac *AttachedCluster) GetSecretName() string {
	return ac.Spec.Kubeconfig.Name
}

func (ac *AttachedCluster) GetSecretKey() string {
	return ac.Spec.Kubeconfig.Key
}

func (ac *AttachedCluster) SetAccepted(accepted bool) {
	ac.Status.Accepted = accepted
}
