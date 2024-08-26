---
title: Single cluster deployment application
authors:
- @yeyeye2333
reviewers:
approvers:

creation-date: 2024-07-24

---

## Single Cluster Deployment Application

### Summary

This proposal aims to allow setting the application's destination to a cluster, reducing the operational steps and enabling users to seamlessly use Kurator in a single-cluster, thereby increasing Kurator's flexibility.

### Motivation

Kurator leverages Fleet for multi-cluster management. However, this has resulted in the current application deployment being tightly coupled with Fleet, even requiring Fleet creation for single-cluster usage. Therefore, we aim to decouple the single-cluster client from Fleet.

#### Goals

- Add a new feature to support setting the application's destination to a cluster.
- Enable users to seamlessly use Kurator in a single-cluster environment, thus eliminating the need for Fleet.

### Proposal

Currently, the application registration is handled in the fleet-manager. The proposal is to add a new feature to deploy the application directly to the current host cluster when ApplicationDestination is not specified.

The preliminary discussion results are as follows:

- If `ApplicationDestination` is not specified, the `application` is deployed directly under the current host cluster, otherwise it is deployed in `ApplicationDestination.fleet`
- When updating the destination configuration, no actions will be taken on applications deployed in the old clusters (no deletion).



### Design Details

This proposal does not involve API modification, and supports new functions in code logic.
In the specific code, if `ApplicationDestination` is empty, `handleSyncPolicyByKind` directly on the current cluster client to create Kustomization or HelmRelease resources.


#### Test Plan

During the coding phase, add unit tests covering core functionalities and edge cases. After completing the coding work, design integration tests to ensure the proper deployment of applications in a single cluster, and subsequently, provide examples of deploying applications in a single cluster.
