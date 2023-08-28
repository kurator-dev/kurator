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
	helmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,categories=kurator-dev
// +kubebuilder:subresource:status

// Application is the schema for the application's API.
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ApplicationSpec   `json:"spec,omitempty"`
	Status            ApplicationStatus `json:"status,omitempty"`
}

// ApplicationSpec defines the configuration to produce an artifact and how to dispatch it.
type ApplicationSpec struct {
	// Source defines the artifact source.
	Source ApplicationSource `json:"source"`
	// SyncPolicies controls how the artifact will be customized and where it will be synced.
	SyncPolicies []*ApplicationSyncPolicy `json:"syncPolicies"`
	// Destination defines the destination clusters where the artifacts will be synced.
	// It can be overriden by the syncPolicies' destination.
	// +optional
	Destination *ApplicationDestination `json:"destination,omitempty"`
}

// ApplicationSource defines the configuration to produce an artifact for git, helm or oci repository.
// Note only one source can be specified.
type ApplicationSource struct {
	// +optional
	GitRepository *sourcev1beta2.GitRepositorySpec `json:"gitRepository,omitempty"`
	// +optional
	HelmRepository *sourcev1beta2.HelmRepositorySpec `json:"helmRepository,omitempty"`
	// +optional
	OCIRepository *sourcev1beta2.OCIRepositorySpec `json:"ociRepository,omitempty"`
}

// ApplicationDestination defines the configuration to dispatch an artifact to a fleet or specific clusters.
type ApplicationDestination struct {
	// Fleet defines the fleet to dispatch the artifact.
	// +required
	Fleet string `json:"fleet"`
	// ClusterSelector specifies the selectors to select the clusters within the fleet.
	// If unspecified, all clusters in the fleet will be selected.
	// +optional
	ClusterSelector *ClusterSelector `json:"clusterSelector,omitempty"`
}

type ClusterSelector struct {
	// MatchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value".
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	// +optional
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// ApplicationSyncPolicy defines the configuration to sync an artifact.
// Only oneof `kustomization` or `helm` can be specified to manage application sync.
type ApplicationSyncPolicy struct {
	// Name defines the name of the sync policy.
	// If unspecified, a name of format `<application name>-<index>` will be generated.
	// +optional
	Name string `json:"name,omitempty"`

	// Kustomization defines the configuration to calculate the desired state
	// from a source using kustomize.
	// +optional
	Kustomization *Kustomization `json:"kustomization,omitempty"`
	// HelmRelease defines the desired state of a Helm release.
	// +optional
	Helm *HelmRelease `json:"helm,omitempty"`

	// Destination defines the destination for the artifact.
	// If specified, it will override the destination specified at Application level.
	// +optional
	Destination *ApplicationDestination `json:"destination"`
}

// ApplicationStatus defines the observed state of Application.
type ApplicationStatus struct {
	SourceStatus *ApplicationSourceStatus `json:"sourceStatus,omitempty"`
	SyncStatus   []*ApplicationSyncStatus `json:"syncStatus,omitempty"`
}

// applicationSourceStatus defines the observed state of the artifact source.
type ApplicationSourceStatus struct {
	GitRepoStatus  *sourcev1beta2.GitRepositoryStatus  `json:"gitRepoStatus,omitempty"`
	HelmRepoStatus *sourcev1beta2.HelmRepositoryStatus `json:"helmRepoStatus,omitempty"`
	OCIRepoStatus  *sourcev1beta2.OCIRepositoryStatus  `json:"ociRepoStatus,omitempty"`
}

// ApplicationSyncStatus defines the observed state of Application sync.
type ApplicationSyncStatus struct {
	Name                string                                `json:"name,omitempty"`
	KustomizationStatus *kustomizev1beta2.KustomizationStatus `json:"kustomizationStatus,omitempty"`
	HelmReleaseStatus   *helmv2beta1.HelmReleaseStatus        `json:"HelmReleaseStatus,omitempty"`
}

// ApplicationList contains a list of Application.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}
