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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,categories=kurator-dev
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.schedule",description="Schedule defines when to run the Backup using a Cron expression.If not set, the backup will be executed only once"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase of the Backup"

// Backup is the schema for the Backup's API.
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupSpec   `json:"spec,omitempty"`
	Status            BackupStatus `json:"status,omitempty"`
}

type BackupSpec struct {
	// Storage details where the backup data should be stored.
	Storage BackupStorage `json:"storage"`

	// Schedule defines when to run the Backup using a Cron expression.
	// If not set, the backup will be executed only once.
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// Destination indicates the default clusters where backups should be executed.
	// Can be overridden by individual Policies.
	// +optional
	Destination *Destination `json:"destination,omitempty"`

	// Policies are the rules defining how backups should be performed.
	// +optional
	Policies []*BackupSyncPolicy `json:"policies,omitempty"`
}

type BackupStorage struct {
	// Location specifies the location where the backup data will be stored.
	Location BackupStorageLocation `json:"location"`

	// Credentials to access the backup storage location.
	Credentials string `json:"credentials"`
}

type BackupStorageLocation struct {
	// Bucket specifies the storage bucket name.
	Bucket string `json:"bucket"`
	// Provider specifies the storage provider type (e.g., aws).
	Provider string `json:"provider"`
	// S3Url provides the endpoint URL for S3-compatible storage.
	S3Url string `json:"s3Url"`
	// Region specifies the region of the storage.
	Region string `json:"region"`
}

type BackupSyncPolicy struct {
	// Name of the BackupSyncPolicy.
	// If not provided, a default name will be generated.
	// This field is recommended for users to set, so that during the restore process, customized restoration can be performed based on this name.
	// +optional
	Name string `json:"name,omitempty"`

	// Destination indicates where the backup should be executed.
	// +optional
	Destination Destination `json:"destination,omitempty"`

	// Policy outlines the specific rules and filters applied during the backup process.
	// It determines which resources are selected for backup and any specific conditions or procedures to follow.
	// Users can customize this policy to ensure that the backup process aligns with their specific requirements and constraints.
	// +optional
	Policy BackupPolicy `json:"policy,omitempty"`
}

// Note: partly copied from https://github.com/vmware-tanzu/velero/pkg/apis/backup_types.go
// BackupSpec defines the specification for a backup.
type BackupPolicy struct {
	// ResourceFilter specifies which resources should be included in the backup.
	// It acts as a selective criterion to determine which resources are relevant for backup.
	// If not set, the backup process will consider all resources. This filter helps in optimizing the backup process by excluding unnecessary data.
	// +optional
	ResourceFilter *ResourceFilter `json:"resourceFilter,omitempty"`

	// TTL is a time.Duration-parseable string describing how long the Backup should be retained for.
	// +optional
	TTL metav1.Duration `json:"ttl,omitempty"`

	// OrderedResources specifies the backup order of resources of specific Kind.
	// The map key is the resource name and value is a list of object names separated by commas.
	// Each resource name has format "namespace/objectname".  For cluster resources, simply use "objectname".
	// +optional
	// +nullable
	OrderedResources map[string]string `json:"orderedResources,omitempty"`

	// ItemOperationTimeout specifies the time used to wait for asynchronous BackupItemAction operations.
	// The default value is 1 hour.
	// +optional
	ItemOperationTimeout metav1.Duration `json:"itemOperationTimeout,omitempty"`
}

type BackupStatus struct {
	// Conditions represent the current state of the backup operation.
	// +optional
	Conditions capiv1.Conditions `json:"conditions,omitempty"`

	// Phase represents the current phase of the backup operation.
	// +optional
	Phase string `json:"phase,omitempty"`

	// BackupDetails provides a detailed status for each backup in each cluster.
	// +optional
	BackupDetails []*velerov1.BackupStatus `json:"backupDetails,omitempty"`
}
