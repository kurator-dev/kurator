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
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,categories=kurator-dev
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase of the Restore"

// Restore is the schema for the Restore's API.
type Restore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RestoreSpec   `json:"spec,omitempty"`
	Status            RestoreStatus `json:"status,omitempty"`
}

type RestoreSpec struct {
	// BackupName specifies the backup on which this restore operation is based.
	// Corresponds to the object name in the CRD "backup.kurator.dev".
	// +required
	BackupName string `json:"backupName"`

	// Destination indicates the clusters where restore should be performed.
	// if not set, all the clusters from Backup.spec.destination will perform restore.
	// +optional
	Destination *Destination `json:"destination,omitempty"`

	// Policy defines the customization rules for the restore.
	// If null, the backup will be fully restored using default settings.
	// +optional
	Policy *RestorePolicy `json:"policy,omitempty"`
}

// Note: partly copied from https://github.com/vmware-tanzu/velero/blob/v1.11.1/pkg/apis/velero/v1/restore_types.go
// RestorePolicy defines the specification for a Velero restore.
type RestorePolicy struct {
	// ResourceFilter is the filter for the resources to be restored.
	// If not set, all resources from the backup will be restored.
	// +optional
	ResourceFilter *ResourceFilter `json:"resourceFilter,omitempty"`

	// NamespaceMapping is a map of source namespace names
	// to target namespace names to restore into.
	// Any source namespaces not included in the map will be restored into
	// namespaces of the same name.
	// +optional
	NamespaceMapping map[string]string `json:"namespaceMapping,omitempty"`

	// PreserveStatus specifies which resources we should restore the status field.
	// If unset, no status will be restored.
	// +optional
	// +nullable
	PreserveStatus *PreserveStatus `json:"preserveStatus,omitempty"`

	// PreserveNodePorts specifies whether to restore old nodePorts from backup.
	// If not specified, default to false.
	// +optional
	// +nullable
	PreserveNodePorts *bool `json:"preserveNodePorts,omitempty"`
}

// PreserveStatus specifies which resources we should restore the status field.
// The logic implemented is that everything in the included list except those items in the excluded list should be included.
// '*' in the includes list means "include everything", but it is not valid in the exclude list.
type PreserveStatus struct {
	// IncludedResources determines which resources will have their status restored.
	// By default, if this list is empty, it means the status for ALL resources will be restored.
	// For example:
	// - If you want to restore the status only for deployments and services, set:
	//   includedResources: ["deployments", "services"]
	// - If you leave it empty, the status for all resources will be restored.
	// Note: If a resource is listed in both IncludedResources and ExcludedResources, the exclusion takes precedence.
	// +optional
	// +nullable
	IncludedResources []string `json:"includedResources,omitempty"`

	// ExcludedResources lists out the resources that will NOT have their status restored.
	// By default, if this list is empty, it means the status for NO resources will be excluded from restoration.
	// For instance:
	// - If you want to prevent restoring the status for pods and configmaps, set:
	//   excludedResources: ["pods", "configmaps"]
	// - If you leave it empty, no resources are excluded, and all will have their status restored (unless specified otherwise in IncludedResources).
	// Note: Exclusions listed here take precedence over inclusions. So, if a resource is listed in both, its status will NOT be restored.
	// +optional
	// +nullable
	ExcludedResources []string `json:"excludedResources,omitempty"`
}

type RestoreStatus struct {
	// Conditions represent the current state of the restore operation.
	// +optional
	Conditions capiv1.Conditions `json:"conditions,omitempty"`

	// Phase represents the current phase of the restore operation.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Details provides a detailed status for each restore in each cluster.
	// +optional
	Details []*RestoreDetails `json:"restoreDetails,omitempty"`
}

type RestoreDetails struct {
	// ClusterName is the Name of the cluster where the restore is being performed.
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// ClusterKind is the kind of ClusterName recorded in Kurator.
	// +optional
	ClusterKind string `json:"clusterKind,omitempty"`

	// RestoreNameInCluster is the name of the restore being performed within this cluster.
	// This RestoreNameInCluster is unique in Storage.
	// +optional
	RestoreNameInCluster string `json:"restoreNameInCluster,omitempty"`

	// RestoreStatusInCluster is the current status of the restore performed within this cluster.
	// +optional
	RestoreStatusInCluster *velerov1.RestoreStatus `json:"restoreStatusInCluster,omitempty"`
}

// RestoreList contains a list of Restore.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Restore `json:"items"`
}
