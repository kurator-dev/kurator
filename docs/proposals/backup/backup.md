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

The proposal aims to enhance Kurator by introducing unified backup, restore, and migration capabilities.

With the increasing complexity and distribution of cloud-native applications, there's a pressing need for a unified system that can efficiently handle backup, restore, and migration tasks. 

By introducing these capabilities in Kurator, we aim to provide users with a robust solution that offers a unified operational view, simplifying the process of backup, restore and migrate applications and data across clusters.

#### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

The following three capabilities only require users to declare the desired API state in one place, and Kurator will automatically handle all subsequent operations, 
including the installation of Velero on each cluster, execution of specific operations, and unified aggregation of the status of each operation.

- **unified backup**
    - Support for individual namespaces, multiple namespaces, or an entire cluster within the fleet.
    - Support for almost any type of Kubernetes volume, such as EFS, AzureFile, NFS, emptyDir, local, or any other volume type.
    - Support for different policy with customization rules and resources filter based on `name`, `namespace` or `label`.
    - Support for apply those policy in multiple clusters or an individual cluster within the fleet.
    - Support for scheduled backup.
- **unified backup**
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
Besides, Snapshot will need tie to a specific storage platform. 
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

The core of this proposal revolves around three primary tasks:

1. **Design of Custom Resource Definitions (CRDs)** Design three distinct CRDs to encapsulate the functionalities of unified backup, restore, and migration:

- Unified Backup: 
    - Storage that details where the backup data should be stored.
    - Options for enabling scheduled backups and defining the associated scheduling strategy.
    - Capability to segment multiple sub-clusters within the fleet arbitrarily (achieved through 'select') and apply different backup strategies for these sub-cluster groups.
- Unified Restore: 
    - Unified backup which Restore based on.
    - Options for restore partial content.
- Unified Migration: 
    - Storage that details where the backup data should be stored.
    - One migration source.
    - One or more migration destination clusters.

1. **Implementation through Fleet-Manager** The fleet-manager will actively monitor these CRDs. Based on user configurations, it will:

- Install Velero on each fleet clusters.
- Execute the functionalities of unified backup, restore, and migration as defined by the CRDs.
- Handle potential errors or exceptions, ensuring smooth operations.

1. **Status Aggregation** The fleet-manager will:

- Aggregate backup and restoration statuses from each cluster, reflecting them within the CRD's status section.
- Summarize migration stages, updating the CRD's status section accordingly.

#### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

##### Story 1

**User Role**: Operations Engineer managing multi-cluster Kubernetes environments

**Scenario**: In a multi-cloud environment, operations engineers need to periodically back up their Kubernetes cluster resources to meet compliance and disaster recovery requirements. Manually backing up each cluster is time-consuming and prone to errors.

**Feature**: Use the automated Velero installation tool, supporting automatic unified backup, restore, and migration settings for multiple clusters in the fleet. Configure backup policies for multiple clusters at once and automatically execute them as scheduled.

**Value**: Provides a streamlined, automated way to manage backup and recovery across multiple clusters uniformly. Reduces human errors, ensuring data continuity and compliance.

**Outcome**: By using this feature, the operations engineer can easily and automatically back up all cluster resources and quickly restore them when needed, ensuring business continuity and data security.

##### Story 2

**User Role**: DevOps Engineer

**Scenario**: In a Continuous Integration and Continuous Deployment (CI/CD) environment, DevOps engineers need to be able to precisely select Kubernetes cluster resources to back up and restore as needed.

**Feature**: Supports scheduled backups and the ability to filter resources for backup and restore based on Type, Namespace, and other conditions.

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
```

Here is the details about `Destination` and `ResourceFilter`

```console
// Destination defines a fleet or specific clusters.
type Destination struct {
	// Fleet is the name of fleet.
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

// Note: partly copied from https://github.com/"github.com/vmware-tanzu/velero/pkg/apis/backup_types.go
type ResourceFilter struct {
	// IncludedNamespaces is a slice of namespace names to include objects from.
	// If empty, all namespaces are included.
	// +optional
	// +nullable
	IncludedNamespaces []string `json:"includedNamespaces,omitempty"`

	// ExcludedNamespaces contains a list of namespaces that are not included in the backup.
	// +optional
	// +nullable
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`

	// IncludedResources is a slice of resource names to include in the backup.
	// If empty, all resources are included.
	// +optional
	// +nullable
	IncludedResources []string `json:"includedResources,omitempty"`

	// ExcludedResources is a slice of resource names that are not included in the backup.
	// +optional
	// +nullable
	ExcludedResources []string `json:"excludedResources,omitempty"`

	// IncludedClusterScopedResources is a slice of cluster-scoped resource type names to include in the backup.
	// If set to "*", all cluster-scoped resource types are included.
	// The default value is empty, which means only related cluster-scoped resources are included.
	// +optional
	// +nullable
	IncludedClusterScopedResources []string `json:"includedClusterScopedResources,omitempty"`

	// ExcludedClusterScopedResources is a slice of cluster-scoped resource type names to exclude from the backup.
	// If set to "*", all cluster-scoped resource types are excluded. The default value is empty.
	// +optional
	// +nullable
	ExcludedClusterScopedResources []string `json:"excludedClusterScopedResources,omitempty"`

	// IncludedNamespaceScopedResources is a slice of namespace-scoped resource type names to include in the backup.
	// The default value is "*".
	// +optional
	// +nullable
	IncludedNamespaceScopedResources []string `json:"includedNamespaceScopedResources,omitempty"`

	// ExcludedNamespaceScopedResources is a slice of namespace-scoped resource type names to exclude from the backup.
	// If set to "*", all namespace-scoped resource types are excluded. The default value is empty.
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

	// IncludeClusterResources specifies whether cluster-scoped resources should be included for consideration in the backup.
	// +optional
	// +nullable
	IncludeClusterResources *bool `json:"includeClusterResources,omitempty"`
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
```

#### Flow Diagrams

To better comprehend the workings of the aforementioned APIs, we provide corresponding flow diagrams. 

These diagrams illustrate the various steps involved in backup, restore, and migration, and how they interact with Kurator and Velero.

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
