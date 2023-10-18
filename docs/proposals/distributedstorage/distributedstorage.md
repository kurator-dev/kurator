---
title: Distributed Storage System for Kurator
authors:
- "@LiZhenCheng9527" # Authors' GitHub accounts here.
reviewers:
approvers:

creation-date: 2023-09-08

---

## Distributed Storage System for Kurator

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

In order to further enhance its functionality, this proposal introduces a unified solution for distributed storage across multiple clusters in `Fleet` through a seamless one-click operation.

By integrating rook, we aim to provide users with reliable, fast and unified distributed storage, enabling them to easily use block, file and object storage in multiple clusters.

### Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users.
-->

In the current era of data explosion, distributed cloud storage has the advantages of easy scalability, high concurrency, reliability, high availability and high storage efficiency.

Kurator is open source distributed cloud native suite, providing users with a one-stop open source solution for distributed cloud native scenarios. So distributed storage as an important part of cloud native usage scenarios, kurator needs to provide relevant functional support.

#### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
Unified distributed cloud storage only requires the user to declare the required API state in one place, Kurator will automatically handle all subsequent operations, will be in the cluster according to the statement in the Spec, different nodes deployed different daemon, using different types of storage. And a unified summary of the status of each operation. In kurator, you can choose to distribute a configuration to a specific single or multiple clusters.

- **unified distributed cloud storage**
    - Support three types of storage: Block storage, Filesystem storage and Object storage.
    - Support for specifying different storage types at different nodes by name, label and annotation.
    - Support for applying storage type policies on nodes to multiple clusters.
  
#### Non-Goals

- **Do not support for another distributed storage system** Rook only supports ceph, so it does not support other distributed storage systems.
- **Do not support for Erasure Code** Erasure Code is a data redundancy protection technique. The ability to recover lost data within certain limits is a more cost-effective way to improve the reliability of storage systems than the three-copy approach. However, Kurator now only supports the three-copy method. It may add related features in the future!

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

### Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->
This proposal aims to introduce unified distributed cloud storage for Kurator that supports Block storage, Filesystem storage and Object storage. The main objectives of this proposal are as follows:

Custom Resource Definitions (CRDs): Design CRDs to enable unified distributed cloud storage capabilities. These CRDs will provide a structured approach for defining clusters, different storage types for implementing distributed cloud storage.

Fleet-Manager Implementation: The Cluster Manager component will be responsible for monitoring the CRDs and performing the defined functions. It will install Rook on the clusters and handle potential errors or anomalies to ensure smooth operation.

By integrating these enhancements, Kurator will provide users with a powerful yet streamlined solution for managing the task of implementing distributed cloud storage and simplifying the overall operational process.

#### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

##### Story 1

**User Role**: Operator managing multi-cluster Kubernetes environments

**Feature**:  With the enhanced Kurator, Operators can easily configure distributed storage policies for multiple clusters simultaneously.

**Value**: Provides a simplified, automated way to unify the management of distributed storage across multiple clusters. Reduces human error and ensures data continuity and compliance.

**Outcome**: Using this feature, Operator can easily configure distributed storage for all clusters to improve the reliability, availability and storage efficiency of business system storage, as well as easy scalability.

##### Story 2

**User Role**: Operator managing many different workload which need different storage type

**Feature**: Operator have the ability to slap different labels on the nodes where different workload reside in Kubernetes. With the enhanced Kurator, based on these labels, Operator are able to specify the appropriate storage type for the corresponding workload easily.

**Outcome**: Using this feature, Operators can provide different types of storage for different workload in all clusters to improve business system storage efficiency and overall service performance.

#### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->
**Resource Required** In order to configure the Ceph storage cluster, at least one of these local storage options are required:

- Raw devices (no partitions or formatted filesystems)
- Raw partitions (no formatted filesystem)
- LVM Logical Volumes (no formatted filesystem)
- Persistent Volumes available from a storage class in block mode

#### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? 

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->
**Version Compatibility** It is recommended that users use a newer version of Kubernetes, as older versions are not tested with Rook. Kubernetes v1.19 or higher is supported by Rook.

### Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

In this section, we will dive into the detailed API designs for the Unified Distributed Cloud Storage feature.

These APIs are designed to facilitate Kurator's integration with Rook to enable the required functionality.

In contrast to Rook, we may need to adapt the Unified Distributed Cloud Storage to reflect our new strategy and decisions.

#### Unified Distributed Storage System API

Kurator is designed to install the Rook as a fleet plugin. In order to achieve unified distributed storage.

Here's the preliminary design for the Unified Distributed Storage:

```console
type PluginConfig struct {
    // Metric defines the configuration for the monitoring system installation and metrics collection..
    // +optional
    Metric *MetricConfig `json:"metric,omitempty"`
    // Grafana defines the configuration for the grafana installation and observation.
    // +optional
    Grafana *GrafanaConfig `json:"grafana,omitempty"`
    // Policy defines the configuration for the ploicy management.
    Policy *PolicyConfig `json:"policy,omitempty"`
    // Backup defines the configuration for the backup engine(Velero).
    Backup *BackupConfig `json:"backup,omitempty"`
    // DistributedStorage define the configuration for the distributed storage(Implemented with Rook)
    DistributedStorage *DistributedStorageConfig `json:"distributedStorage,omitempty"`
}

type DistributedStorageConfig struct {
    // Chart defines the helm chart configuration of the distributed storage engine.
    // The default value is:
    //
    // chart:
    //   repository: https://charts.rook.io/release
    //   name: rook
    //   version: 1.11.11
    //
    // +optional
    Chart *ChartConfig `json:"chart,omitempty"`

    // Storage provides detailed settings for unified distributed storage.
    Storage DistributedStorage `json:"storage"`

    // ExtraArgs provides the extra chart values for rook chart.
    // For example, use the following configuration to change the pull policy:
    //
    // extraArgs:
    //   image:
    //     pullPolicy: Always
    //
    // +optional
    ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}

// Note: partly copied from https://github.com/rook/rook/blob/release-1.10/pkg/apis/ceph.rook.io/v1/types.go
type DistributedStorage struct {
    // The path on the host where config and data can be persisted.
    // If the storagecluster is deleted, please clean up the configuration files in this file path.
    // e.g. /var/lib/rook
    // +kubebuilder:validation:Pattern=`^/(\S+)`
    // +optional
    DataDirHostPath *string `json:"dataDirHostPath,omitempty"`

    // Monitor is the daemon that monitors the status of the ceph cluster.
    // Responsible for collecting cluster information, updating cluster information, and publishing cluster information.
    // Including monmap, osdmap, PGmap, mdsmap, etc. 
    // A spec for mon related options
    // +optional
    // +nullable
    Monitor *MonSpec `json:"monitor,omitempty"`

    // Manager is the daemon runs alongside monitor daemon,to provide additional monitoring and interfaces to external monitoring and management systems.
    // A spec for mgr related options
    // +optional
    // +nullable
    Manager *MgrSpec `json:"manager,omitempty"`

    // A spec for available storage in the cluster and how it should be used
    // +optional
    // +nullable
    Storage *StorageScopeSpec `json:"storage,omitempty"`
}
```

Here is the details about `Monitor` and `Manager`

Monitor's count is recommended to be an odd number.
Since the monitor is responsible for collecting, updating, and publishing cluster information in the ceph cluster, every time you read or write data, you need to use the monitor to get the mapping information of the stored data. Therefore, multiple copies are needed to improve data availability. However, multiple copies will introduce data consistency problems. We use the paxos algorithm to ensure data consistency in the ceph cluster. paxos algorithm requires more than half of the monitors in the ceph cluster to be active.

When the number of monitors is 3, it requires 2 active monitors to work properly; when the number of monitors is 4, it requires 3 active monitors to work properly. So the disaster tolerance of an odd number of monitors is the same as the disaster tolerance of an even number of monitors whose number + 1. Therefore, it is recommended to set the number of monitors in the cluster to an odd number.

```console
// Note: partly copied from https://github.com/rook/rook/blob/release-1.10/pkg/apis/ceph.rook.io/v1/types.go

type MonSpec struct {
    // Count is the number of Ceph monitors.
    // Default is three and preferably an odd number.
    // +kubebuilder:validation:Minimum=0
    // +kubebuilder:validation:Maximum=9
    // +optional
    Count int `json:"count,omitempty"`
    
    // In a ceph cluster, it is recommended that the monitor pod be deployed on a different node in order to ensure high availability of data.
    // In practice, you can label the node where the monitor pod is deployed with Annotation/Labels.
    // Then use kubernetes node affinity rules to achieve the goal of deploying the monitor to different nodes.
    // The annotations-related configuration to add/set on each Pod related object.
    // +nullable
    // +optional
    Annotations rookv1.AnnotationsSpec `json:"annotations,omitempty"`

    // Similar to Annotation, but more graphical than Annotation.
    // The labels-related configuration to add/set on each Pod related object.
    // +kubebuilder:pruning:PreserveUnknownFields
    // +nullable
    // +optional
    Labels rookv1.LabelsSpec `json:"labels,omitempty"`

    // The placement-related configuration to pass to kubernetes (affinity, node selector, tolerations).
    // +kubebuilder:pruning:PreserveUnknownFields
    // +nullable
    // +optional
    Placement rookv1.PlacementSpec `json:"placement,omitempty"`
}

type MgrSpec struct {
    // Count is the number of manager to run
    // Default is two, one for use and one for standby.
    // +kubebuilder:validation:Minimum=0
    // +kubebuilder:validation:Maximum=2
    // +optional
    Count int `json:"count,omitempty"`

    // Use Annotations/labels to achieve the goal of placing two managers on different nodes.
    // The annotations-related configuration to add/set on each Pod related object.
    // +nullable
    // +optional
    Annotations rookv1.AnnotationsSpec `json:"annotations,omitempty"`

    // The labels-related configuration to add/set on each Pod related object.
    // +kubebuilder:pruning:PreserveUnknownFields
    // +nullable
    // +optional
    Labels rookv1.LabelsSpec `json:"labels,omitempty"`

    // The placement-related configuration to pass to kubernetes (affinity, node selector, tolerations).
    // +kubebuilder:pruning:PreserveUnknownFields
    // +nullable
    // +optional
    Placement rookv1.PlacementSpec `json:"placement,omitempty"`
}
```

Here is the details about `StorageScopeSpec`

```console
// Note: partly copied from https://github.com/rook/rook/blob/release-1.10/pkg/apis/ceph.rook.io/v1/types.go
type StorageScopeSpec struct {
    // +nullable
    // +optional
    Nodes []Node `json:"nodes,omitempty"`
    
    // indicating if all nodes in the cluster should be used for storage according to the cluster level storage selection and configuration values. 
    // If individual nodes are specified under the nodes field, then useAllNodes must be set to false.
    // +optional
    UseAllNodes bool `json:"useAllNodes,omitempty"`
   
    // Select device information used by osd. For more information see the design of the selection below.
    StorageDeviceSelection `json:",inline"`

    // OSDStore is the backend storage type used for creating the OSDs
    // Default OSDStore type is bluestore which can directly manages bare devices
    // +optional
    Store rookv1.OSDStore `json:"store,omitempty"`
}

// Each individual node can specify configuration to override the cluster level settings and defaults. 
// If a node does not specify any configuration then it will inherit the cluster level settings.
type Node struct {
    // Name should match its kubernetes.io/hostname label
    // +optional
    Name string `json:"name,omitempty"`
    
    // Specify which storage drives the osd deployed in this node can manage.
    // +kubebuilder:pruning:PreserveUnknownFields
    // +nullable
    // +optional
    StorageDeviceSelection `json:",inline"`
}

// This type of cluster can specify devices for OSDs, both at the cluster and individual node level, for selecting which storage resources will be included in the cluster.
// These selected devices do not mean that they need to be on all nodes, but osd will use them for storage. 
// If these settings are not available, osd will also run on the specified nodes and listen for the status of the storage devices on the nodes. 
// Once a specified device is plugged into a node, osd formats and plugs that device into osd for use.
// More info please refer to https://github.com/rook/rook/blob/master/Documentation/Getting-Started/quickstart.md#prerequisites
type StorageDeviceSelection struct {
    // List of devices to use as storage devices
    // A list of individual device names belonging to this node to include in the storage cluster
    // e.g. `sda` or  `/dev/disk/by-id/ata-XXXX`
    // +kubebuilder:pruning:PreserveUnknownFields
    // +nullable
    // +optional
    Devices []rookv1.Device `json:"devices,omitempty"`
}
```

Here is a simple example of StorageClusterSpec.

```console
StorageClusterSpec:
    dataDirHostPath: /var/lib/rook
    monitor:
        count: 3
        #To control where monitor will be scheduled by kubernetes, use the placement configuration sections below.
        labels:
            role: MonitorNode
        placement:
            nodeAffinity:
                requiredDuringSchedulingIgnoredDuringExecution:
                    nodeSelectorTerms:
                    - matchExpressions:
                      - key: role
                        operator : NotIn
                        value:
                        - MonitorNode    
    manager:
        count: 2
        #To control where manager will be scheduled by kubernetes, use the placement configuration sections below.
        labels:
            role: ManagerNode
        placement:
            nodeAffinity:
                requiredDuringSchedulingIgnoredDuringExecution:
                    nodeSelectorTerms:
                    - matchExpressions:
                      - key: role
                        operator : NotIn
                        value:
                        - ManagerNode
    storage:
        # Cluster-level configuration, used by nodes not specifically specified in the configuration.
        # The specially designated nodes use their own configuration, as shown below.
        useAllDevices: true
        nodes:
          - name: "172.17.4.201"
            devices:
              - name: sda
              - name: sdb
          - name: "172.17.4.101"
            devices:
              - name: /dev/sdc
          - name: worker3
            devices:
              - name: "nvme01"
```

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

End-to-End Tests: Comprehensive E2E testing should be performed to ensure that block, file, and object storage in distributed storage operate seamlessly across clusters.

Integration Tests: Integration tests should be designed to ensure Kurator's integration with Rook functions as expected.

Unit Tests: Unit tests should cover the core functionalities and edge cases.

Isolation Testing: The distributed storage functionalities should be tested in isolation and in conjunction with other components to ensure compatibility and performance.

### Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

<!--
Note: This is a simplified version of kubernetes enhancement proposal template.
https://github.com/kubernetes/enhancements/tree/3317d4cb548c396a430d1c1ac6625226018adf6a/keps/NNNN-kep-template
-->

The main alternatives considered were other distributed storage systems. For example, openebs or longhorn. but if you want to use ceph, you have to use a Rook. Considering that Ceph is more advanced than other storage systems due to its decentralised and CRUSH nature. Therefore, the Rook should be used in Kurator.
