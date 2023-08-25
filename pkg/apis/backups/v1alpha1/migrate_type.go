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
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase of the Migrate"

// Migrate is the schema for the Migrate's API.
type Migrate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MigrateSpec   `json:"spec,omitempty"`
	Status            MigrateStatus `json:"status,omitempty"`
}

type MigrateSpec struct {
	// Storage details where the data should be stored.
	Storage BackupStorage `json:"storage"`

	// SourceCluster represents the source cluster for migration.
	SourceCluster *Destination `json:"sourceCluster"`

	// TargetCluster represents the target clusters for migration.
	TargetCluster *Destination `json:"targetCluster"`

	// Policy defines the rules for the migration.
	// +optional
	Policy *MigratePolicy `json:"policy,omitempty"`
}

type MigratePolicy struct {
	// ResourceFilter specifies the resources to be included in the migration.
	// If not set, all resources in  will be migrated.
	// +optional
	ResourceFilter *ResourceFilter `json:"resourceFilter,omitempty"`

	// OrderedResources specifies the backup order of resources of specific Kind.
	// The map key is the resource name and value is a list of object names separated by commas.
	// Each resource name has format "namespace/objectname".  For cluster resources, simply use "objectname".
	// +optional
	// +nullable
	OrderedResources map[string]string `json:"orderedResources,omitempty"`

	// NamespaceMapping is a map of source namespace names to target namespace names to migrate into.
	// Any source namespaces not included in the map will be migrated into namespaces of the same name.
	// +optional
	NamespaceMapping map[string]string `json:"namespaceMapping,omitempty"`

	// MigrateStatus specifies which resources we should migrate the status field.
	// If nil, no objects are included. Optional.
	// +optional
	// +nullable
	MigrateStatus *RestoreStatusSpec `json:"migrateStatus,omitempty"`

	// PreserveNodePorts specifies whether to migrate old nodePorts from source cluster to target cluster.
	// +optional
	// +nullable
	PreserveNodePorts *bool `json:"preserveNodePorts,omitempty"`

	// ItemOperationTimeout specifies the time used to wait for RestoreItemAction operations.
	// The default value is 1 hour.
	// +optional
	ItemOperationTimeout metav1.Duration `json:"itemOperationTimeout,omitempty"`
}

type MigrateStatus struct {
	// Conditions represent the current state of the migration operation.
	// +optional
	Conditions capiv1.Conditions `json:"conditions,omitempty"`

	// Phase represents the current phase of the migration operation.
	// +optional
	Phase string `json:"phase,omitempty"`

	// SourceClusterStatus provides a detailed status for backup in SourceCluster.
	SourceClusterStatus *velerov1.BackupStatus `json:"sourceClusterStatus,omitempty"`

	// TargetClusterStatus provides a detailed status for each restore in each TargetCluster.
	TargetClusterStatus []*velerov1.RestoreStatus `json:"targetClusterStatus,omitempty"`
}
