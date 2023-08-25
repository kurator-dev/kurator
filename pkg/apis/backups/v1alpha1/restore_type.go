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
	BackupName string `json:"backupName"`

	// Policies defines the customization rules for the restore.
	// If null, the backup will be fully restored using default settings.
	// +optional
	Policies []*RestoreSyncPolicy `json:"policies,omitempty"`
}

type RestoreSyncPolicy struct {
	// Name is the unique identifier for this restore policy. It should match the name in the backup policy.
	// to ensure the restore policy corresponds to the correct backup policy.
	// If a name provided by the user doesn't match any backup policy, the restore operation will fail
	// and return a clear error message.
	Name string `json:"name"`

	// Policy indicates the rules and filters for the restore.
	// +optional
	Policy RestorePolicy `json:"policy,omitempty"`
}

// Note: partly copied from https://github.com/vmware-tanzu/velero/pkg/apis/restore_types.go
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

	// RestoreStatus specifies which resources we should restore the status field.
	// If nil, no objects are included.
	// +optional
	// +nullable
	RestoreStatus *RestoreStatusSpec `json:"restoreStatus,omitempty"`

	// PreserveNodePorts specifies whether to restore old nodePorts from backup.
	// +optional
	// +nullable
	PreserveNodePorts *bool `json:"preserveNodePorts,omitempty"`

	// ItemOperationTimeout specifies the time used to wait for RestoreItemAction operations.
	// The default value is 1 hour.
	// +optional
	ItemOperationTimeout metav1.Duration `json:"itemOperationTimeout,omitempty"`
}

type RestoreStatusSpec struct {
	// IncludedResources specifies the resources to which will restore the status.
	// If empty, it applies to all resources.
	// +optional
	// +nullable
	IncludedResources []string `json:"includedResources,omitempty"`

	// ExcludedResources specifies the resources to which will not restore the status.
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

	// RestoreDetails provides a detailed status for each restore in each cluster.
	// +optional
	RestoreDetails []*velerov1.RestoreStatus `json:"restoreDetails,omitempty"`
}
