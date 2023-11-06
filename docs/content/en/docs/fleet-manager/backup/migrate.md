---
title: "Unified Migration"
linkTitle: "Unified Migration"
weight: 40
description: >
  Effortlessly migrate resources across clusters using a single configuration.
---

Unified Migration simplifies the process of transferring resources between different clusters.
With a simple configuration, application resources can be transferred across multiple clusters simultaneously. 
This guide will detail the streamlined steps, empowering you with the knowledge and tools essential for unified migration.

## Introduction

## Migration Overview

- **Concept:** Unified Migration is about transferring application and resources from one cluster to several other clusters.

- **Configuration:** Users only need to define a `migrate` type of resource. This configuration encapsulates source cluster, origin clusters, and further policies.

- **Automation:** Upon applying the resource configuration, `FleetManager` in Kurator host autonomously manages tasks, starting from uploading resources from the original cluster to object storage, and finally, transferring them to the desired target destination clusters, as shown in the diagram below:

{{< image width="100%"
link="./image/migrate.svg"
>}}

- **Flexibility:** Depending on user-defined settings and strategies, the migration procedure will slightly differ, offering a bespoke experience for each use case.

- **Unified Monitoring view:** Users can monitor the migration progress across multiple clusters from a single point of reference — the 'migrate' object status. This centralized view provides real-time insights, allowing users to keep a pulse on the migration status and ensure that everything is proceeding as expected.

## How to Perform a Unified Migration

### Pre-requisites

Before diving into the migration steps, ensure that:

- You have successfully installed the backup plugin as described in the [backup plugin installation guide](/docs/fleet-manager/backup/backup-plugin).
- You have correctly configured `fleet` and `attachedcluster` based on the instructions from the previous guide.
- You have ensured that `kurator-member1` has the cluster resources labeled with `app:busybox`.

### Steps to Follow

**Execute Migration Configuration:** Begin the migration process based on the defined configuration.

```console
kubectl apply -f examples/backup/migrate-select-labels.yaml
```

**Monitor the Migration Progress:** To keep track of the migration process, use the following command:

```console
kubectl get migrates.backup.kurator.dev select-labels -o yaml
```

**Verification:** After a brief wait, you should notice that the resources from `kurator-member1` have been successfully migrated to `kurator-member2`.

**Confirmation:** Once the migration is successful, executing the command `kubectl get migrates.backup.kurator.dev select-labels -o yaml` should yield results similar to the output provided below.

```console
apiVersion: backup.kurator.dev/v1alpha1
kind: Migrate
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"backup.kurator.dev/v1alpha1","kind":"Migrate","metadata":{"annotations":{},"name":"select-labels","namespace":"default"},"spec":{"policy":{"resourceFilter":{"labelSelector":{"matchLabels":{"app":"busybox"}}}},"sourceCluster":{"clusters":[{"kind":"AttachedCluster","name":"kurator-member1"}],"fleet":"quickstart"},"targetCluster":{"clusters":[{"kind":"AttachedCluster","name":"kurator-member2"}],"fleet":"quickstart"}}}
  creationTimestamp: "2023-10-28T15:55:13Z"
  finalizers:
  - migrate.kurator.dev
  generation: 1
  name: select-labels
  namespace: default
  resourceVersion: "9511446"
  uid: c948154d-1727-4d01-bf47-7ffb03f800a3
spec:
  policy:
    resourceFilter:
      labelSelector:
        matchLabels:
          app: busybox
  sourceCluster:
    clusters:
    - kind: AttachedCluster
      name: kurator-member1
    fleet: quickstart
  targetCluster:
    clusters:
    - kind: AttachedCluster
      name: kurator-member2
    fleet: quickstart
status:
  conditions:
  - lastTransitionTime: "2023-10-28T15:55:23Z"
    status: "True"
    type: sourceReady
  phase: Completed
  sourceClusterStatus:
    backupNameInCluster: kurator-member1-migrate-default-select-labels
    backupStatusInCluster:
      completionTimestamp: "2023-10-28T15:55:18Z"
      expiration: "2023-11-27T15:55:13Z"
      formatVersion: 1.1.0
      phase: Completed
      progress: {}
      startTimestamp: "2023-10-28T15:55:13Z"
      version: 1
    clusterKind: AttachedCluster
    clusterName: kurator-member1
  targetClusterStatus:
  - clusterKind: AttachedCluster
    clusterName: kurator-member2
    restoreNameInCluster: kurator-member2-migrate-default-select-labels
    restoreStatusInCluster:
      completionTimestamp: "2023-10-28T15:56:00Z"
      phase: Completed
      startTimestamp: "2023-10-28T15:55:58Z"
```

Upon inspecting the details of the `Migrate` object, we can infer the following:

- **Specification (`spec`):**
    - **Source Cluster:** Resources are being migrated from `kurator-member1`.
    - **Target Cluster:** The destination for the migration is `kurator-member2`.
    - **Migration Policy:** The policy for this migration is to transfer resources that have the label `app: busybox`.

- **Status (`status`):**
    - **`phase`:** The overall status or `phase` of the migration process is `Completed`. This indicates that the migration was successful.
    - **`sourceClusterStatus`:** This section provides backup details about the source cluster `kurator-member1`. 
    - **`targetClusterStatus`:** This section provides restore details about the target cluster `kurator-member2`. 

### Cleanup

To remove the migration examples used for testing, execute:

```console
kubectl delete migrates.backup.kurator.dev select-labels
```

> Please note: This command only deletes the current object in the k8s API. 
Application or resources from both before and after the migration, as well as data in the object storage, will not be deleted
