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
	// SourceCluster represents the source cluster for migration.
	// The user needs to ensure that SourceCluster points to only ONE cluster.
	// Because the current migration only supports migrating from one SourceCluster to one or more TargetCluster.
	// +required
	SourceCluster Destination `json:"sourceCluster"`

	// TargetClusters represents the target clusters for migration.
	// +required
	TargetClusters Destination `json:"targetCluster"`

	// Policy defines the rules for the migration.
	Policy *MigratePolicy `json:"policy,omitempty"`
}

type MigratePolicy struct {
	// ResourceFilter specifies the resources to be included in the migration.
	// If not set, all resources in source cluster will be migrated.
	// +optional
	ResourceFilter *ResourceFilter `json:"resourceFilter,omitempty"`

	// OrderedResources specifies the backup order of resources of specific Kind.
	// The map key is the resource name and value is a list of object names separated by commas.
	// Each resource name has format "namespace/objectname".  For cluster resources, simply use "objectname".
	// For example, if you have a specific order for pods, such as "pod1, pod2, pod3" with all belonging to the "ns1" namespace,
	// and a specific order for persistentvolumes, such as "pv4, pv8", you can use the orderedResources field in YAML format as shown below:
	//
	// ```yaml
	// orderedResources:
	//  pods: "ns1/pod1, ns1/pod2, ns1/pod3"
	//  persistentvolumes: "pv4, pv8"
	// ```
	//
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
	MigrateStatus *PreserveStatus `json:"migrateStatus,omitempty"`

	// PreserveNodePorts specifies whether to migrate old nodePorts from source cluster to target cluster.
	// +optional
	// +nullable
	PreserveNodePorts *bool `json:"preserveNodePorts,omitempty"`
}

// MigratePhase is a string representation of the lifecycle phase of a Migrate instance
// +kubebuilder:validation:Enum=New;FailedValidation;WaitingForSource;InProgress;Completed;Failed
type MigratePhase string

const (
	// MigratePhaseNew means the migrate has been created but not
	// yet processed by the RestoreController
	MigratePhaseNew MigratePhase = "New"

	// MigratePhaseFailedValidation means the migrate has failed
	// the controller's validations and therefore will not run.
	MigratePhaseFailedValidation MigratePhase = "FailedValidation"

	// MigratePhaseWaitingForSource means the migrate is currently fetching source cluster resource.
	MigratePhaseWaitingForSource MigratePhase = "WaitingForSource"

	// MigratePhaseSourceReady means the migrate is already currently fetched source cluster resource.
	MigratePhaseSourceReady MigratePhase = "SourceReady"

	// MigratePhaseInProgress means the migrate is currently executing migrating.
	MigratePhaseInProgress MigratePhase = "InProgress"

	// MigratePhaseCompleted means the migrate has run successfully
	// without errors.
	MigratePhaseCompleted MigratePhase = "Completed"

	// MigratePhaseFailed means the migrate was unable to execute.
	MigratePhaseFailed MigratePhase = "Failed"
)

type MigrateStatus struct {
	// Conditions represent the current state of the migration operation.
	// +optional
	Conditions capiv1.Conditions `json:"conditions,omitempty"`

	// Phase represents the current phase of the migration operation.
	// +optional
	Phase MigratePhase `json:"phase,omitempty"`

	// SourceClusterStatus provides a detailed status for backup in SourceCluster.
	SourceClusterStatus *BackupDetails `json:"sourceClusterStatus,omitempty"`

	// TargetClusterStatus provides a detailed status for each restore in each TargetCluster.
	TargetClustersStatus []*RestoreDetails `json:"targetClusterStatus,omitempty"`
}

// MigrateList contains a list of Migrate.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MigrateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Migrate `json:"items"`
}
