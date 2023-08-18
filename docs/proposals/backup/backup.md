---
title: Unified Backup, Restore, and Migration with Velero in Kurator
authors:
- "@Xieql"
reviewers:
- "@robot"
- TBD
approvers:
- "@robot"
- TBD

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

By introducing these capabilities in Kurator, we aim to provide users with a robust solution that offers a unified operational view, simplifying the process of migrating applications and data across clusters.

#### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Automatically install Velero for clusters in the fleet.
- Support automated unified backup for either multiple clusters or an individual cluster within the fleet.
- Support automated unified restore for either multiple clusters or an individual cluster within the fleet
- Support automated unified migration for either multiple clusters or an individual cluster within the fleet.
- Support automated unified scheduled backup; If the current backups are not scheduled, allow users to easily convert it into scheduled backup.
- Support filtering resources for backup, restore and migration based on type, namespace or other conditions.
- User can view the current execution status of all backups, restores, and migrations from a single location.

#### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Limit the development and testing environment to on-premise clusters and [Kind](https://kind.sigs.k8s.io/). Besides, the Object Storage Service(OSS) is limited to [Minio](https://min.io/docs/minio/kubernetes/upstream/).
- Provide only the [Restic](https://github.com/restic/restic) solution for storage involving Persistent Volumes due to the limitations of snapshot-based solutions in cross-cluster functionality. See [velero doc](https://velero.io/doc)
- Basically, focus solely on the initial configuration, excluding subsequent configuration edit or reapply.

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
    - Parameters used during Velero installation for backup.
    - Options for enabling scheduled backups and defining the associated scheduling strategy.
    - Capability to segment multiple sub-clusters within the fleet arbitrarily (achieved through 'select') and apply different backup strategies for these sub-cluster groups.
- Unified Restore: 
    - Unified backup which Restore based on.
    - Options for restore partial content.
- Unified Migration: 
    - Parameters used during Velero installation for backup.
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


- **Restic Limitations**
Restic does not support the hostpath PV, which means it cannot be tested in a kind cluster. 
If a backup is attempted with a hostpath type, Velero will skip this resource and continue with the subsequent resources. Reference [velero issue](https://github.com/vmware-tanzu/velero/issues/4962)

- **Testing with Kind**
When testing with the kind cluster, it's recommended to use the busybox example instead of the nginx example provided by Velero.

- **Backup Frequency**
If the backup frequency is set too short, with the default backup retention period of 30 days, it might lead to a large amount of data in the OSS, potentially causing system crashes. 
It's advisable to highlight this in the documentation for users.

- **Velero Readiness**
Kurator needs to ensure that Velero is ready on each cluster (including the ability to connect to the specified OSS) before proceeding with further operations.

- **Velero Version Differences**
There are significant differences in fields between Velero versions before and after 1.10. 

- **Local Cluster Testing**
For local cluster testing, if there's no storage class (SC) available, it's essential to install an SC to ensure the busybox example with PV runs correctly. 
It's recommended to set the PV type to local instead of hostpath.




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
As mentioned, different versions of Velero have different fields. 
There's a risk of compatibility issues if clusters are running different versions. It's crucial to ensure all clusters run a supported version of Velero. 
The most recent version of Velero is 1.12, and it has been tested exclusively with versions ranging from 1.25.7 to 1.27.3.


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

##### Unified Backup API

Here's the preliminary design for the Unified Backup API:

```console
apiVersion: backups.kurator.dev/v1alpha1
kind: FleetBackup
metadata:
  name: testBackup
  namespace: default
spec:
  schedule:
    open: true
    cron: 0 0 * * *
  destination:
    fleet: quickstart
  veleroInstall:
    xxx
  backupPolicies:
    - destination:
        clusterSelector:
          matchLabels:
            env: test
      backupPolicy:
        xxx
    - destination:
        clusterSelector:
          matchLabels:
            env: dev
      backupPolicy:
        xxx
status:
  - clusterName:
    clusterBackupStatus:
  - clusterName:
    clusterBackupStatus:
  - clusterName:
    clusterBackupStatus:
```



##### Unified Restore API

Below is the initial design for the Unified Restore API:

```console
apiVersion: backups.kurator.dev/v1alpha1
kind: FleetRestore
metadata:
  name: testRestore
  namespace: default
spec:
  fleetBackup: testBackup
  restorePolicies:
    - destination:
        clusterSelector:
          matchLabels:
            policy: policy1
      restorePolicy:
        xxx
    - destination:
        clusterSelector:
          matchLabels:
            policy: policy2
      restorePolicy:
        xxx
status:
  - clusterName:
    clusterRestoreStatus:
  - clusterName:
    clusterRestoreStatus:
  - clusterName:
    clusterRestoreStatus:
```

##### Unified Migration API

Presenting the initial design for the Unified Migration API:

```console
apiVersion: backups.kurator.dev/v1alpha1
kind: FleetMigration
metadata:
  name: testMigration
  namespace: default
spec:
  originCluster:
  destination:
    clusterSelector:
      matchLabels:
        target: target1
  # same as backupPolicy
  migrationPolicies
    xxx
status:
  migrationStatus:
  - originClusterName:
    originClustereStatus:
  - destinationClusterName:
    destinationClusterStatus:
  - destinationClusterName:
    destinationClusterStatus:
```

#### Flow Diagrams

To better comprehend the workings of the aforementioned APIs, we provide corresponding flow diagrams. 

These diagrams illustrate the various steps involved in backup, restore, and migration, and how they interact with Kurator and Velero.

##### Backup Flow Diagram

This is the sequence diagram for unified backup.

{{< image width="100%"
    link="./image/backup.svg"
    >}}

##### Restore Flow Diagram

The flow for unified restore is quite similar with unified backup, with the main difference being that it involves restoring from the OSS instead of performing a backup.

##### Migration Flow Diagram

Here's the flow diagram for unified migration.

{{< image width="100%"
    link="./image/migration.svg"
    >}}

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
