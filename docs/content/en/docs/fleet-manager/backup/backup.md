---
title: "Unified Backup"
linkTitle: "Unified Backup"
weight: 20
description: >
  A comprehensive guide on Kurator's Unified Backup solution, providing an overview and practical implementation steps.
---

Unified Backup offers a concise, one-click solution to back up application and cluster resources across all clusters in Fleet,
supporting varying granularities from individual namespaces to multiple namespaces, and up to entire clusters.
This ensures consistent data protection and facilitates swift recovery in a distributed cloud-native infrastructure.

## Introduction

### Backup Types

Unified Backup supports two primary backup methods to cater to different requirements, ensuring data is safely stored and can be restored when necessary.

#### Immediate Backup

- **Use Case**: Essential for sudden changes, like after a significant data update or before implementing system alterations.
- **Functionality**: Triggers a one-time backup on demand.

#### Scheduled Backup

- **Use Case**: Regularly back up dynamic data to safeguard against accidental losses, maintain compliance, and enable efficient disaster recovery.
- **Functionality**: Set automated backups at specific intervals using cron expressions.

### Advanced Backup Options

- **Specific Cluster Backup within Fleet**: Users can choose one or more particular clusters within the Fleet for backup.
- **Resource Filtering**: Kurator offers filtering options for more precise backups, allowing users to define criteria based on attributes like name, namespace, or label.

> **Note**: Kurator supports backing up and restoring Kubernetes volumes attached to pods directly from the file system. Snapshot support is currently unavailable. Refer to the documentation [FSB](https://velero.io/docs/v1.11/file-system-backup/) for more information.

## How to Perform a Unified Backup


In the subsequent sections, we'll guide you through a hands-on demonstration. 
Before delving into the details, ensure you have successfully installed the backup plugin as outlined in the  [backup plugin installation guide](/docs/fleet-manager/backup/backup-plugin).


For this demonstration, we will be using the `fleet` and two [kind](https://kind.sigs.k8s.io/) clusters, as created in the  [backup plugin installation guide](/docs/fleet-manager/backup/backup-plugin). 

You can initiate the process by deploying a test application using the following command:

```console
kubectl apply -f examples/backup/app-backup-demo.yaml 
```

Executing the above command will deploy the test busybox across two clusters within the `fleet`. For a comprehensive understanding about `app-backup-demo.yaml`, please refer to the [unified application distribution](/docs/fleet-manager/application).

Due to constraints associated with `restic`, performing a PV backup in the kind cluster (the environment we're currently operating in) is a challenging [issue](https://github.com/vmware-tanzu/velero/issues/4962). 
Hence, our subsequent examples will mainly feature `busybox`. However, in a real-world cluster setup, users can opt for the `busybox-with-pv` example.

### 1. Performing an Immediate Backup

Apply an immediate backup example:

```console
kubectl apply -f examples/backup/backup-select-labels.yaml
```

Review the results:

```console
kubectl get backups.backup.kurator.dev select-labels -o yaml
```

The expected result should be:

```console
apiVersion: backup.kurator.dev/v1alpha1
kind: Backup
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"backup.kurator.dev/v1alpha1","kind":"Backup","metadata":{"annotations":{},"name":"select-labels","namespace":"default"},"spec":{"destination":{"fleet":"quickstart"},"policy":{"resourceFilter":{"labelSelector":{"matchLabels":{"app":"busybox"}}},"ttl":"720h"}}}
  creationTimestamp: "2023-10-28T03:37:07Z"
  finalizers:
  - backup.kurator.dev
  generation: 1
  name: select-labels
  namespace: default
  resourceVersion: "9380210"
  uid: e412fdea-d0d8-43a9-9f24-3d48333dc0a3
spec:
  destination:
    fleet: quickstart
  policy:
    resourceFilter:
      labelSelector:
        matchLabels:
          app: busybox
    ttl: 720h
status:
  backupDetails:
  - backupNameInCluster: kurator-member1-backup-default-select-labels
    backupStatusInCluster:
      completionTimestamp: "2023-10-28T03:37:13Z"
      expiration: "2023-11-27T03:37:07Z"
      formatVersion: 1.1.0
      phase: Completed
      progress:
        itemsBackedUp: 1
        totalItems: 1
      startTimestamp: "2023-10-28T03:37:07Z"
      version: 1
    clusterKind: AttachedCluster
    clusterName: kurator-member1
  - backupNameInCluster: kurator-member2-backup-default-select-labels
    backupStatusInCluster:
      completionTimestamp: "2023-10-28T03:37:13Z"
      expiration: "2023-11-27T03:37:07Z"
      formatVersion: 1.1.0
      phase: Completed
      progress: {}
      startTimestamp: "2023-10-28T03:37:07Z"
      version: 1
    clusterKind: AttachedCluster
    clusterName: kurator-member2
```

Given the output provided, let's dive deeper to understand the various elements and their implications:

- In the spec, the `destination` field is used. By default, if no specific cluster is set, it points to all clusters within the `fleet`.
- The `policy` defines the backup strategy. Using the `resourceFilter`, it specifies that the backup should target resources with the label `app: busybox`. For more advanced filtering options, refer to the [Fleet API](https://kurator.dev/docs/references/fleet-api/#fleet)
- The `status` section displays the actual processing status of the two clusters within the fleet. 

Furthermore, you can check your backup data in object storage; it will appear in the bucket used during the plugin configuration.

### 2. Performing a Scheduled Backup

Apply the scheduled backup:

```console
kubectl apply -f examples/backup/backup-schedule.yaml
```

Review the results:

```console
kubectl get backups.backup.kurator.dev schedule -o yaml
```

The expected result should be:

```console
apiVersion: backup.kurator.dev/v1alpha1
kind: Backup
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"backup.kurator.dev/v1alpha1","kind":"Backup","metadata":{"annotations":{},"name":"schedule","namespace":"default"},"spec":{"destination":{"clusters":[{"kind":"AttachedCluster","name":"kurator-member2"}],"fleet":"quickstart"},"schedule":"*/5 * * * *"}}
  creationTimestamp: "2023-10-28T02:38:39Z"
  finalizers:
  - backup.kurator.dev
  generation: 1
  name: schedule
  namespace: default
  resourceVersion: "9369767"
  uid: a7dd8dc1-2a56-424f-ad11-e55ec434043e
spec:
  destination:
    clusters:
    - kind: AttachedCluster
      name: kurator-member2
    fleet: quickstart
  schedule: '*/5 * * * *'
status:
  backupDetails:
  - backupStatusInCluster: {}
    clusterKind: AttachedCluster
    clusterName: kurator-member2
```

Analyzing the provided output, let's dissect its sections for a clearer comprehension:

- **Cron Expression in `spec`**:
    - The `schedule` field within the `spec` uses a cron expression. This defines when a backup is to be performed. 
    - The expression '*/5 * * * *' means the backup will run every 5 minutes. This setting is for testing purposes only. Users should adjust this parameter based on their actual needs.
    - Once set, the backup won't be executed immediately. Instead, it waits until the time specified by the cron expression.

- **Destination in `spec`**:
    - The `destination` field under `spec` points to `kurator-member2`. This means the backup is specifically for this cluster within its fleet.

- **Policy in `spec`**:
    - The `policy` section outlines the backup strategy. If no `policy` specified, all cluster resource will be backup.
    - For more advanced filtering options, refer to the [Fleet API](https://kurator.dev/docs/references/fleet-api/#fleet)
  
- **Status Section**:
    - The `status` section provides an overview of the backup status across clusters.
    - At the moment, it's empty. As backups are executed according to the cron schedule, this section will populate with relevant details.

### Cleanup

To remove the backup examples used for testing, execute:

```console
kubectl delete backups.backup.kurator.dev select-labels schedule
```

> Please note: This command only deletes the current object in the k8s API.
For data security, deletion of the data in object storage should be performed using the tools provided by the object storage solution you adopted. 
Kurator currently does not offer capabilities to delete data inside the object storage.
