---
title: Unified Backup, Restore, and Migration with Velero in Kurator
authors:
- "@Xieql"
reviewers:
approvers:

creation-date: 2023-08-18

---

## Unified Backup, Restore, and Migration with Velero in Kurator

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

### Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. 

A good summary is probably at least a paragraph in length.
-->

Kurator, as an open-source distributed cloud-native platform, has been pivotal in aiding users to construct their distributed cloud-native infrastructure, thereby facilitating enterprise digital transformation. 

To further enhance its capabilities, this proposal introduces a unified solution for backup, restore, and migration across multiple clusters in `Fleet` through a seamless one-click operation. 

By integrating Velero, we aim to provide a unified operational view, simplifying the backup and restoration process and facilitating easy migration across clusters.

### Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users.
-->

In a multi-cloud environment, operators face the challenge of regularly backing up their Kubernetes cluster resources to comply with compliance and disaster recovery requirements. 
Manual backups for each cluster are time-consuming and error-prone.

Meanwhile, in Continuous Integration and Continuous Deployment (CI/CD) environments, DevOps engineers need precise control over selecting and backing up specific Kubernetes cluster resources as per their needs.

To address these challenges, this proposal aims to enhance Kurator by introducing unified backup, restore, and migration capabilities.

By incorporating these capabilities into Kurator, we aim to provide users with a robust and streamlined solution. 
This will offer a unified operational view, simplifying the process of backing up, restoring, and migrating applications and data across clusters.

#### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

The following three capabilities only require users to declare the desired API state in one place, and Kurator will automatically handle all subsequent operations, 
including the installation of Velero on each cluster, execution of specific operations, and unified aggregation of the status of each operation.

- **unified backup**
    - Support for individual namespaces, multiple namespaces, or an entire cluster.
    - Support for almost any type of Kubernetes volume, such as EFS, AzureFile, NFS, emptyDir, local and so on.
    - Support for different policy with customization rules and resources filter based on `name`, `namespace` or `label`.
    - Support for apply those policy in multiple clusters or an individual cluster.
    - Support for scheduled backup.
- **unified restore**
    - Support for restore all resource from backup.
    - Support for partly restore by resources filter for each backup policy.
- **unified migrate**
    - Support for migrate the cluster' resource from one source cluster to one or multi target clusters.
    - Support for customization rules and resources filter.


#### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- **support for Volume Snapshot** Many commonly used types, such as EFS and NFS, do not have a native snapshot concept. 
Besides, Snapshot will need tie to a specific storage platform.  "Velero does not natively support the migration of persistent volumes snapshots across cloud providers" see [Velero doc](https://velero.io/docs/v1.12/migration-case/)
Moreover,  we currently lack the conditions to test Snapshot.
- **support for velero advanced features** Initially, we will not implement the advanced features like hook capability in our integration.
We will decide on adding this feature in the future based on user feedback and requirements.


### Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

This proposal aims to introduce unified backup, restore, and migration capabilities into Kurator. The main objectives of this proposal are as follows:

Custom Resource Definitions (CRDs): Design three CRDs to encompass the functionalities of unified backup, restore, and migration. These CRDs will provide a structured approach for defining storage locations, scheduling backups, restoring partial content, and specifying migration sources and destinations.

Fleet-Manager Implementation: The fleet-manager component will be responsible for monitoring the CRDs and executing the defined functionalities. It will install Velero on fleet clusters and handle potential errors or exceptions to ensure smooth operations.

By incorporating these enhancements, Kurator will offer users a robust and streamlined solution for managing backup, restore, and migration tasks, simplifying the overall operational process

#### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

##### Story 1

**User Role**: Operator managing multi-cluster Kubernetes environments

**Feature**: With the enhanced Kurator, Operator can easily configure backup policies for multiple clusters simultaneously and schedule automated backups.

**Value**: Provides a streamlined, automated way to manage backup and recovery across multiple clusters uniformly. Reduces human errors, ensuring data continuity and compliance.

**Outcome**: By using this feature, the operations engineer can easily and automatically back up all cluster resources and quickly restore them when needed, ensuring business continuity and data security.

##### Story 2

**User Role**: DevOps Engineer

**Feature**: The enhanced Kurator offers advanced functionality such as flexible resource filtering options based on Type, Namespace, and other conditions allows engineer to precisely choose the resources to back up, restore and migrate cluster resource as needed.

**Value**: Ability to flexibly choose backup resources and precisely restore needed resources, making the DevOps process more efficient and flexible.

**Outcome**: By using this feature, the DevOps engineer can flexibly back up and restore resources according to the needs of the CI/CD process, supporting faster and more efficient software delivery.

#### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

- **hostPath volumes will be skipped when backup**
If we want backup an application with hostPath volumes, we need change hostPath type to local type, 
or Velero will skip this resource with hostPath volumes and continue with the subsequent resources.
It is Worth mentioning is that Kind cluster use hostPath volumes. see [Velero issue](https://github.com/vmware-tanzu/velero/issues/4962)

- **Backup Frequency**
If the backup frequency is set too short, with the default backup retention period of 30 days, it might lead to a large amount of data in the OSS, potentially causing system crashes. 
It's advisable to highlight this in the documentation for users.

- **Velero Readiness**
Kurator needs to ensure that Velero is ready on each cluster (including the ability to connect to the specified OSS) before proceeding with further operations.


#### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? 

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

- **Data Integrity**
As with any backup and restore solution, there's always a risk of data corruption or loss. 
It is necessary to remind users to ensure that the resources are in a normal state and not being edited when performing backups.
By integrating Velero, we aim to minimize this risk, but it's essential to have regular checks and validations.

- **Version Compatibility**
It is recommended that users use a newer version of Kubernetes, as older versions are not tested with Velero. 
For example, in the latest version of Velero, versions prior to 1.25 have not been tested.

- **Resource Limitations**
Intensive backup operations might strain the resources of the OSS or the clusters.


### Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

#### API Design

In this section, we delve into the detailed API designs for the unified backup, restore, and migration functionalities. 
These API designs facilitate Kurator's integration with Velero to achieve the desired functionalities.

Compared to Velero, we might need to make adjustments to the Unified Backup API, Unified Restore API, and Unified Migration API to reflect our new strategies and decisions.

##### Unified Backup API

Here's the preliminary design for the Unified Backup API:

```console
// Backup is the schema for the Backup's API.
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BackupSpec   `json:"spec,omitempty"`
	Status            BackupStatus `json:"status,omitempty"`
}

type BackupSpec struct {
	// TODO: consider add Storage setting for backup

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
	// +optional
	Destination *Destination `json:"destination,omitempty"`

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

	// TODO: consider SnapshotVolumes for backup

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
```

Here is the details about `Destination` and `ResourceFilter`

```console

// Destination defines a fleet or specific clusters.
type Destination struct {
	// Fleet is the name of fleet.
	// The field, in combination with ClusterSelector, can determine a set of clusters.
	// In addition to this approach, users can also directly specify clusters through the field Clusters.
	// +optional
	Fleet string `json:"fleet,omitempty"`

	// ClusterSelector specifies the selectors to select the clusters within the fleet.
	// If unspecified, all clusters in the fleet will be selected.
	// The field will only take effect when Fleet is set.
	// +optional
	ClusterSelector *ClusterSelector `json:"clusterSelector,omitempty"`

	// Clusters determine a set of clusters as destination clusters.
	// The field will only take effect when Fleet is not set.
	// +optional
	Clusters []*corev1.ObjectReference `json:"clusters,omitempty"`
}

type ClusterSelector struct {
	// MatchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value".
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	// +optional
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// Note: partly copied from https://github.com/vmware-tanzu/velero/blob/v1.11.1/pkg/apis/velero/v1/backup_types.go
type ResourceFilter struct {
	// IncludedNamespaces is a list of namespace names to include objects from.
	// If empty, all namespaces are included.
	// +optional
	// +nullable
	IncludedNamespaces []string `json:"includedNamespaces,omitempty"`

	// ExcludedNamespaces contains a list of namespaces that are not included in the backup.
	// +optional
	// +nullable
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`

	// IncludedResources is a slice of API resource names to include in the backup.
	// For example, we can populate this string array with "deployments" and "configmaps", then we will select all resources of type deployments and configmaps.
	// If empty, all API resources are included.
	// Cannot work with IncludedClusterScopedResources, ExcludedClusterScopedResources, IncludedNamespaceScopedResources and ExcludedNamespaceScopedResources.
	// +optional
	// +nullable
	IncludedResources []string `json:"includedResources,omitempty"`

	// ExcludedResources is a slice of resource names that are not included in the backup.
	// Cannot work with IncludedClusterScopedResources, ExcludedClusterScopedResources, IncludedNamespaceScopedResources and ExcludedNamespaceScopedResources.
	// +optional
	// +nullable
	ExcludedResources []string `json:"excludedResources,omitempty"`

	// IncludeClusterResources specifies whether cluster-scoped resources should be included for consideration in the backup.
	// Cannot work with IncludedClusterScopedResources, ExcludedClusterScopedResources, IncludedNamespaceScopedResources and ExcludedNamespaceScopedResources.
	// +optional
	// +nullable
	IncludeClusterResources *bool `json:"includeClusterResources,omitempty"`

	// IncludedClusterScopedResources is a slice of cluster-scoped resource type names to include in the backup.
	// For example, we can populate this string array with "storageclasses" and "clusterroles", then we will select all resources of type storageclasses and clusterroles,
	// If set to "*", all cluster-scoped resource types are included.
	// The default value is empty, which means only related cluster-scoped resources are included.
	// Cannot work with IncludedResources, ExcludedResources and IncludeClusterResources.
	// +optional
	// +nullable
	IncludedClusterScopedResources []string `json:"includedClusterScopedResources,omitempty"`

	// ExcludedClusterScopedResources is a slice of cluster-scoped resource type names to exclude from the backup.
	// If set to "*", all cluster-scoped resource types are excluded. The default value is empty.
	// Cannot work with IncludedResources, ExcludedResources and IncludeClusterResources.
	// +optional
	// +nullable
	ExcludedClusterScopedResources []string `json:"excludedClusterScopedResources,omitempty"`

	// IncludedNamespaceScopedResources is a slice of namespace-scoped resource type names to include in the backup.
	// For example, we can populate this string array with "deployments" and "configmaps", then we will select all resources of type deployments and configmaps,
	// The default value is "*".
	// Cannot work with IncludedResources, ExcludedResources and IncludeClusterResources.
	// +optional
	// +nullable
	IncludedNamespaceScopedResources []string `json:"includedNamespaceScopedResources,omitempty"`

	// ExcludedNamespaceScopedResources is a slice of namespace-scoped resource type names to exclude from the backup.
	// If set to "*", all namespace-scoped resource types are excluded. The default value is empty.
	// Cannot work with IncludedResources, ExcludedResources and IncludeClusterResources.
	// +optional
	// +nullable
	ExcludedNamespaceScopedResources []string `json:"excludedNamespaceScopedResources,omitempty"`

	// LabelSelector is a metav1.LabelSelector to filter with when adding individual objects to the backup.
	// If empty or nil, all objects are included. Optional.
	// +optional
	// +nullable
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// OrLabelSelectors is list of metav1.LabelSelector to filter with when adding individual objects to the backup.
	// If multiple provided they will be joined by the OR operator.
	// LabelSelector as well as OrLabelSelectors cannot co-exist in backup request, only one of them can be used.
	// +optional
	// +nullable
	OrLabelSelectors []*metav1.LabelSelector `json:"orLabelSelectors,omitempty"`
}
```


The Velero support **Global Scope Resource Filtering**: When specific ns-scope resources are designated, if their descriptions involve cluster-scope resources, 
Velero will automatically back up these essential resources without requiring users to configure them separately. However, users can still manually configure the global scope resource filtering.


##### Unified Restore API

Below is the initial design for the Unified Restore API:

```console
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

	// Destination indicates the clusters where restore should be performed.
	// +optional
	Destination *Destination `json:"destination,omitempty"`

	// Policy defines the customization rules for the restore.
	// If null, the backup will be fully restored using default settings.
	// +optional
	Policy *RestorePolicy `json:"policy,omitempty"`
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

	// ReserveStatus specifies which resources we should restore the status field.
	// If unset, no status will be restored.
	// +optional
	// +nullable
	ReserveStatus  *ReserveStatusSpec `json:"restoreStatus,omitempty"`

	// PreserveNodePorts specifies whether to restore old nodePorts from backup.
	// 
	// +optional
	// +nullable
	PreserveNodePorts *bool `json:"preserveNodePorts,omitempty"`
}

type ReserveStatusSpec struct {
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
```

##### Unified Migration API

Presenting the initial design for the Unified Migration API:

```console
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
}

type MigrateStatus struct {
	// Conditions represent the current state of the migration operation.
	// +optional
	Conditions capiv1.Conditions `json:"conditions,omitempty"`

	// Phase represents the current phase of the migration operation.
	// +optional
	Phase string `json:"phase,omitempty"`

	// SourceClusterStatus provides a detailed status for backup in SourceCluster.
	SourceClusterStatus *BackupDetails `json:"sourceClusterStatus,omitempty"`

	// TargetClusterStatus provides a detailed status for each restore in each TargetCluster.
	TargetClusterStatus []*RestoreDetails `json:"targetClusterStatus,omitempty"`
}

// MigrateList contains a list of Migrate.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MigrateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Migrate `json:"items"`
}
```

#### Flow Diagrams

To better comprehend the workings of the aforementioned APIs, we provide corresponding flow diagrams. 

These diagrams illustrate the various steps involved in backup, restore, and migration, and how they interact with Kurator and Velero.

##### Technical Architecture

This is the technical architecture.

![backup-struct](./image/backup-struct.svg)


##### Backup Flow Diagram


This is the sequence diagram for unified backup.

![backup](./image/backup.svg)

##### Restore Flow Diagram

The flow for unified restore is quite similar with unified backup, with the main difference being that it involves restoring from the OSS instead of performing a backup.

##### Migration Flow Diagram

Here's the flow diagram for unified migration.

![migrate](./image/migrate.svg)

#### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

-->


End-to-End Tests: Comprehensive E2E tests should be conducted to ensure the backup, restore, and migration processes work seamlessly across different clusters.

Integration Tests: Integration tests should be designed to ensure Kurator's integration with Velero functions as expected.

Unit Tests: Unit tests should cover the core functionalities and edge cases.

Isolation Testing: The backup, restore, and migration functionalities should be tested in isolation and in conjunction with other components to ensure compatibility and performance.


### Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

The primary alternative considered was to have a unified application distribution approach, where only FluxCD needs to be installed on the Kurator host machine. 
However, due to the absence of cluster-specific fields like kubeconfig in Velero objects, this approach was deemed unfeasible. 
As a result, Velero needs to be installed on each cluster separately, ensuring each cluster's unique configurations are catered to.

<!--
Note: This is a simplified version of kubernetes enhancement proposal template.
https://github.com/kubernetes/enhancements/tree/3317d4cb548c396a430d1c1ac6625226018adf6a/keps/NNNN-kep-template
-->
