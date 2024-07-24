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

This proposal aims to modify the application API to allow setting the application's destination to a cluster, reducing the operational steps required for Kurator in a single-cluster environment and increasing Kurator's flexibility.

### Motivation

Kurator leverages Fleet for multi-cluster management. However, this has resulted in the current application deployment being tightly coupled with Fleet, even requiring Fleet creation for single-cluster usage. Therefore, we aim to decouple the single-cluster client from Fleet.

#### Goals

- Modify the application API to support setting the application's destination to a cluster.
- Simplify the deployment process for single-cluster environments by eliminating the need for Fleet.

### Proposal

Currently, the application registration is handled in the fleet-manager. The proposal is to move the application registration to the cluster-operator to support setting the application's destination to a cluster.

### Design Details

We will delve into the API design required to support setting the application's destination to a cluster. The following are the specific changes to the application:

```go
// ApplicationDestination defines the configuration to dispatch an artifact to a fleet or specific clusters.
// Fleet and cluster are mutually exclusive.
type ApplicationDestination struct {
	// Fleet defines the fleet to dispatch the artifact.
	// +optional
	Fleet *FleetInfo `json:"fleetInfo,omitempty"`

	// Cluster specifies a cluster to dispatch the artifact to.
	// +optional
	Cluster *corev1.ObjectReference `json:"clusters,omitempty"`
}

type FleetInfo struct {
	// Name identifies the name of the fleet.
	// +required
	Name string `json:"name"`
	// ClusterSelector specifies the selectors to select the clusters within the fleet.
	// If unspecified, all clusters in the fleet will be selected.
	// +optional
	ClusterSelector *ClusterSelector `json:"clusterSelector,omitempty"`
}
```

The preliminary discussion results are as follows:

- The destination can only be either a fleet or a cluster. The `ApplicationWebhook.validate()` will check if only one of fleet or cluster is specified.
- When updating the destination configuration, no actions will be taken on applications deployed in the old clusters (no deletion).

In the specific code, replace all `fleet *fleetapi.Fleet` type parameters in the `Reconcile` method with `destinationObject interface{}`. In functions like `fetchClusterList` (previously `fetchFleetClusterList`) that need to parse `destinationObject`, use a `switch destinationObject.(type)` statement for different logical processing to improve code reusability.

#### Test Plan

During the coding phase, add unit tests covering core functionalities and edge cases. After completing the coding work, design integration tests to ensure the proper deployment of applications in a single cluster, and subsequently, provide examples of deploying applications in a single cluster.
