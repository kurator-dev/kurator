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
	// Schedule defines when to run the Backup using a Cron expression.
	// A cron expression is a format used to specify the execution time of recurring tasks, consisting of multiple fields representing different time units.
	// ┌───────────── minute (0 - 59)
	// │ ┌───────────── hour (0 - 23)
	// │ │ ┌───────────── day of the month (1 - 31)
	// │ │ │ ┌───────────── month (1 - 12)
	// │ │ │ │ ┌───────────── day of the week (0 - 6) (Sunday to Saturday;
	// │ │ │ │ │                                   7 is also Sunday on some systems)
	// │ │ │ │ │
	// │ │ │ │ │
	// * * * * *
	// For example, "30 * * * *" represents execution at the 30th minute of every hour, and "10 10,14 * * *" represents execution at 10:10 AM and 2:10 PM every day.
	// If not set, the backup will be executed only once.
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// Destination indicates the clusters where backups should be performed.
	// +required
	Destination Destination `json:"destination"`

	// Policy are the rules defining how backups should be performed.
	// +optional
	Policy *BackupPolicy `json:"policy,omitempty"`
}

// Note: partly copied from https://github.com/vmware-tanzu/velero/blob/v1.11.1/pkg/apis/velero/v1/backup_types.go
// BackupPolicy defines the specification for a backup policy.
type BackupPolicy struct {
	// ResourceFilter specifies which resources should be included in the backup.
	// It acts as a selective criterion to determine which resources are relevant for backup.
	// If not set, the backup process will consider all resources. This filter helps in optimizing the backup process by excluding unnecessary data.
	// +optional
	ResourceFilter *ResourceFilter `json:"resourceFilter,omitempty"`

	// TODO: support volume snapshot

	// TTL is a time.Duration-parseable string describing how long the Backup should be retained for.
	// +optional
	TTL metav1.Duration `json:"ttl,omitempty"`

	// OrderedResources specifies the backup order of resources of specific Kind.
	// The map key is the resource name and value is a list of object names separated by commas.
	// Each resource name has format "namespace/objectname".  For cluster resources, simply use "objectname".
	// For example, if you have a specific order for pods, such as "pod1, pod2, pod3" with all belonging to the "ns1" namespace,
	// and a specific order for persistentvolumes, such as "pv4, pv8", you can use the orderedResources field in YAML format as shown below:
	// ```yaml
	// orderedResources:
	//  pods: "ns1/pod1, ns1/pod2, ns1/pod3"
	//  persistentvolumes: "pv4, pv8"
	// ```
	// +optional
	// +nullable
	OrderedResources map[string]string `json:"orderedResources,omitempty"`
}

type BackupStatus struct {
	// Conditions represent the current state of the backup operation.
	// +optional
	Conditions capiv1.Conditions `json:"conditions,omitempty"`

	// Phase represents the current phase of the backup operation.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Details provides a detailed status for each backup in each cluster.
	// +optional
	Details []*BackupDetails `json:"backupDetails,omitempty"`
}

type BackupDetails struct {
	// ClusterName is the Name of the cluster where the backup is being performed.
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// ClusterKind is the kind of ClusterName recorded in Kurator.
	// +optional
	ClusterKind string `json:"clusterKind,omitempty"`

	// BackupNameInCluster is the name of the backup being performed within this cluster.
	// This BackupNameInCluster is unique in Storage.
	// +optional
	BackupNameInCluster string `json:"backupNameInCluster,omitempty"`

	// BackupStatusInCluster is the current status of the backup performed within this cluster.
	// +optional
	BackupStatusInCluster *velerov1.BackupStatus `json:"backupStatusInCluster,omitempty"`
}

// BackupList contains a list of Backup.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items"`
}
