---
title: "Unified Restore"
linkTitle: "Unified Restore"
weight: 30
description: >
 A comprehensive guide on Kurator's Unified Restore solution, detailing the methodology and steps for data recovery in a distributed cloud-native infrastructure.
---

Unified Restore offers a streamlined method to restore application and cluster resources across all clusters in Fleet, based on backups produced using the Unified Backup approach. 
This feature aids in recovery, aiming for reduced downtime and operational effectiveness.

## Introduction

## Restore Overview

Kurator's Unified Restore feature is designed around the Unified Backup object. 
Based on the type of backup chosen, either Immediate or Scheduled, the restoration method is determined accordingly.

### Restore from an **Immediate Backup**

- **Use Case**: Responding to sudden data losses or application issues.
- **Referred Backup**: The specific **Immediate Backup** designated by the user.
- **Restore Result**: Restoration from the selected backup into designated clusters.

### Restore from a **Scheduled Backup**

- **Use Case**: Restoring to a recent state in scenarios such as post-accidental data modifications, compliance verifications, or disaster recovery after unforeseen system failures.
- **Referred Backup**: When a **Scheduled Backup** is selected, Kurator will automatically target the latest successful backup within that series.
- **Restore Result**: Restoration using the latest successful backup to maintain data relevance and integrity.

### Advanced Backup Options

#### Specific Cluster Restore within Fleet:

Users can specify clusters as the restore destination.
However, these selected clusters must be a subset of those included in the backup.
This is because the restore process relies on the data from the backup.

**Note**: To restore resources from one cluster to another not in the original backup, utilize the Unified Migration feature, detailed in a later section.

#### Resource Filtering:
Users can apply a secondary filter to the data from the backup, enabling selective restoration.
Define the scope using attributes like backup name, namespace, or label to ensure only desired data is restored. For details, refer to the [Fleet API](https://kurator.dev/docs/references/fleet-api/#fleet)

## How to Perform a Unified Restore

### Pre-requisites

Before diving into the restore steps, ensure that:

- You have successfully installed the backup plugin as described in the [backup plugin installation guide](/docs/fleet-manager/backup/backup-plugin).
- You have correctly configured `fleet` and `attachedcluster` based on the instructions from the previous guide.

**Note:** The examples provided in this section correspond directly with those outlined in the [unified backup](/docs/fleet-manager/backup/restore) documentation.

### Steps to Follow

1. **Backup Creation**: This step involves setting up a backup using the existed config. More details can be found in [unified backup](/docs/fleet-manager/backup/restore)

2. **Disaster Simulation**: This involves activities that represent scenarios of data losses or application disruptions.

3. **Restore Execution**: Based on the previously created backup, execute restore config.

4. **Restore Object Review**: This involves examining the details about restore object.

Throughout these operations, users can check the pod's status using the commands below. 
This will help in verifying the initial state of the pod, confirming if the pod was lost due to the simulated disaster, and ascertaining if the pod was restored through unified recovery:

```console
kubectl get po -n kurator-backup --kubeconfig=/root/.kube/kurator-member1.config
kubectl get po -n kurator-backup --kubeconfig=/root/.kube/kurator-member2.config
```

### 1. Restore from an Immediate Backup

**Immediate Backup Creation**

Deploy a test application using the following command:

```console
kubectl apply -f examples/backup/app-backup-demo.yaml 
```

Trigger an immediate backup using the command:

```console
kubectl apply -f examples/backup/backup-select-labels.yaml
```

**Disaster Simulation**

Simulate a disaster event that deletes all previously installed resources using the command:

```console
kubectl delete applications.apps.kurator.dev unified-backup-demo 
```

**Restore Execution**

Apply a restore action that refers to the backup created earlier:

```console
k apply -f examples/backup/restore-minimal.yaml
```

After executing, you'll observe that the `busybox` pod is restored in two clusters.


**Examine the Restore Object**

Use the following command:

```console
kubectl get restores.backup.kurator.dev minimal -o yaml
```

You can expect the output to resemble the provided structure. 
The status section displays the processing status for the two clusters within the fleet.

```console
apiVersion: backup.kurator.dev/v1alpha1
kind: Restore
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"backup.kurator.dev/v1alpha1","kind":"Restore","metadata":{"annotations":{},"name":"minimal","namespace":"default"},"spec":{"backupName":"select-labels"}}
  creationTimestamp: "2023-10-28T09:24:05Z"
  finalizers:
  - restore.kurator.dev
  generation: 1
  name: minimal
  namespace: default
  resourceVersion: "9441827"
  uid: 6cf6154b-4a68-4431-8366-354bd3cb6250
spec:
  backupName: select-labels
status:
  restoreDetails:
  - clusterKind: AttachedCluster
    clusterName: kurator-member1
    restoreNameInCluster: kurator-member1-restore-default-minimal
    restoreStatusInCluster:
      completionTimestamp: "2023-10-28T09:24:07Z"
      phase: Completed
      progress:
        itemsRestored: 2
        totalItems: 2
      startTimestamp: "2023-10-28T09:24:05Z"
  - clusterKind: AttachedCluster
    clusterName: kurator-member2
    restoreNameInCluster: kurator-member2-restore-default-minimal
    restoreStatusInCluster:
      completionTimestamp: "2023-10-28T09:24:07Z"
      phase: Completed
      progress:
        itemsRestored: 2
        totalItems: 2
      startTimestamp: "2023-10-28T09:24:05Z"
```

**Interpreting the Output:**

- **The spec `destination` field:** The absence of a `destination` field indicates that the restore process will occur on all clusters where the backup was executed. Consequently, the `busybox` pod is successfully restored on both clusters.

- **The spec `policy` field:** The absence of a `policy` field means that the restore strategy is entirely based on the initial backup without any secondary filtering.

- **The `status` section:** This provides insights into the processing status of the two clusters.

### 2. Restore from a Scheduled Backup

**Scheduled Backup Creation**

Deploy a test application using the following command:

```console
kubectl apply -f examples/backup/app-backup-demo.yaml 
```

Trigger a schedule backup using the command:

```console
kubectl apply -f examples/backup/backup-schedule.yaml
```

> Please note: since scheduled backups aren't executed immediately, you'll need to wait at least 5 minutes for a backup to complete before proceeding with the subsequent steps.

```console
kubectl get backups.backup.kurator.dev schedule -o yaml
```

Simulate a disaster event that deletes all previously installed resources using the command:

```console
kubectl delete applications.apps.kurator.dev unified-backup-demo 
```

**Restore Execution**

Apply a restore action that refers to the backup created earlier:

```console
kubectl apply -f examples/backup/restore-schedule.yaml
```

After executing, you'll observe that the `busybox` pod is only restored in the second cluster.

**Examine the Restore Object**

Use the following command:

```console
kubectl get restores.backup.kurator.dev schedule -o yaml
```

You can expect the output to resemble the provided structure.
The status section displays the processing status for the two clusters within the fleet.

```console
apiVersion: backup.kurator.dev/v1alpha1
kind: Restore
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"backup.kurator.dev/v1alpha1","kind":"Restore","metadata":{"annotations":{},"name":"schedule","namespace":"default"},"spec":{"backupName":"schedule","destination":{"clusters":[{"kind":"AttachedCluster","name":"kurator-member2"}],"fleet":"quickstart"},"policy":{"resourceFilter":{"labelSelector":{"matchLabels":{"app":"busybox"}}}}}}
  creationTimestamp: "2023-10-28T13:27:03Z"
  finalizers:
  - restore.kurator.dev
  generation: 1
  name: schedule
  namespace: default
  resourceVersion: "9485496"
  uid: 1eab7122-a6bc-4b64-8196-b512078abfa0
spec:
  backupName: schedule
  destination:
    clusters:
    - kind: AttachedCluster
      name: kurator-member2
    fleet: quickstart
  policy:
    resourceFilter:
      labelSelector:
        matchLabels:
          app: busybox
status:
  restoreDetails:
  - clusterKind: AttachedCluster
    clusterName: kurator-member2
    restoreNameInCluster: kurator-member2-restore-default-schedule
    restoreStatusInCluster:
      completionTimestamp: "2023-10-28T13:27:10Z"
      phase: Completed
      progress: {}
      startTimestamp: "2023-10-28T13:27:03Z"
```

**Interpreting the Output:**

Notice that the backup referenced here is "schedule", which backed up all cluster resources. However, the restore here did not use the "minimal" method to execute the default full restore strategy. 
Instead, a second filter was applied during the restore phase.

- **The spec `destination` field:** Here, only the second cluster is specified, so the resources won't be restored in all clusters, but only in the second one.

- **The spec `policy` field:** It specifies `app: busybox`, so only resources with this label will be restored.

- **The `status` section:** As it's a scheduled backup, the status here shows that the current restore is targeting the most recent successful backup. This provides insights into the processing status of restore.

### Cleanup

To remove the backup examples used for testing, execute:

```console
kubectl delete backups.backup.kurator.dev specific-ns schedule-matchlabels
```

> Please note: This command only deletes the current object in the k8s API.
For data security, deletion of the data in object storage should be performed using the tools provided by the object storage solution you adopted.
Kurator currently does not offer capabilities to delete data inside the object storage.