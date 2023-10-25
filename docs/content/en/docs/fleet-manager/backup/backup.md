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


In the following sections, we will present a hands-on demonstration. 
Before proceeding with the content, ensure you have successfully installed the backup plugin as per the [backup plugin installation guide](/docs/fleet-manager/backup/backup-plugin).


For our demonstration, we'll utilize the fleet and two kind clusters which created in [backup plugin installation guide](/docs/fleet-manager/backup/backup-plugin). 

Let's start by deploying a busybox example:

```console
kubectl apply -f examples/backup/busybox.yaml --kubeconfig=/root/.kube/kurator-member1.config
kubectl apply -f examples/backup/busybox.yaml --kubeconfig=/root/.kube/kurator-member2.config
```

### 1. Performing an Immediate Backup

Apply an immediate backup example:

```console
kubectl apply -f examples/backup/backup-specific-ns.yaml
```

Review the results:

```console
kubectl get backups.backup.kurator.dev specific-ns -o yaml
```

The expected result should be:

```console
apiVersion: backup.kurator.dev/v1alpha1
kind: Backup
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"backup.kurator.dev/v1alpha1","kind":"Backup","metadata":{"annotations":{},"name":"specific-ns","namespace":"default"},"spec":{"destination":{"fleet":"quickstart"},"policy":{"resourceFilter":{"includedNamespaces":["kurator-backup"]},"ttl":"720h"}}}
  creationTimestamp: "2023-10-24T12:04:38Z"
  finalizers:
  - backup.kurator.dev
  generation: 1
  name: specific-ns
  namespace: default
  resourceVersion: "8448571"
  uid: 211a8aec-a8fa-4b73-a48c-ebe4ceb75e2f
spec:
  destination:
    fleet: quickstart
  policy:
    resourceFilter:
      includedNamespaces:
      - kurator-backup
    ttl: 720h
status:
  backupDetails:
  - backupNameInCluster: kurator-member1-backup-default-specific-ns
    backupStatusInCluster:
      expiration: "2023-11-23T12:04:38Z"
      formatVersion: 1.1.0
      phase: InProgress
      startTimestamp: "2023-10-24T12:04:38Z"
      version: 1
    clusterKind: AttachedCluster
    clusterName: kurator-member1
  - backupNameInCluster: kurator-member2-backup-default-specific-ns
    backupStatusInCluster:
      expiration: "2023-11-23T12:04:38Z"
      formatVersion: 1.1.0
      phase: InProgress
      startTimestamp: "2023-10-24T12:04:38Z"
      version: 1
    clusterKind: AttachedCluster
    clusterName: kurator-member2
```

Given the output provided, let's dive deeper to understand the various elements and their implications:

- In the spec, the `destination` field is used. By default, if no specific cluster is set, it points to all clusters within the `fleet`.
- The `policy` provides a unified strategy for the backup. The current setting of `resourceFilter` indicates the backup of all resources under the specified namespace `kurator-backup`. The policy used here only touches upon the namespace. For more advanced filtering options, refer to the [Fleet API](https://kurator.dev/docs/references/fleet-api/#fleet)
- The `status` section displays the actual processing status of the two clusters within the fleet. 

Furthermore, you can check your backup data in object storage; it will appear in the bucket used during the plugin configuration.

### 2. Performing a Scheduled Backup

Label the pod intended for backup to facilitate subsequent label-based selections:

```console
kubectl label po busybox env=test -n kurator-backup --kubeconfig=/root/.kube/kurator-member2.config
```

Apply the scheduled backup:

```console
kubectl apply -f examples/backup/backup-schedule.yaml
```

Review the results:

```console
kubectl get backups.backup.kurator.dev schedule-matchlabels -o yaml
```

The expected result should be:

```console
apiVersion: backup.kurator.dev/v1alpha1
kind: Backup
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"backup.kurator.dev/v1alpha1","kind":"Backup","metadata":{"annotations":{},"name":"schedule-matchlabels","namespace":"default"},"spec":{"destination":{"clusters":[{"kind":"AttachedCluster","name":"kurator-member2"}],"fleet":"quickstart"},"policy":{"resourceFilter":{"labelSelector":{"matchLabels":{"env":"test"}}},"ttl":"240h"},"schedule":"0 0 * * *"}}
  creationTimestamp: "2023-10-25T02:55:30Z"
  finalizers:
  - backup.kurator.dev
  generation: 1
  name: schedule-matchlabels
  namespace: default
  resourceVersion: "8606170"
  uid: 4ac4eb1c-e2cf-48a2-b197-87397d72222a
spec:
  destination:
    clusters:
    - kind: AttachedCluster
      name: kurator-member2
    fleet: quickstart
  policy:
    resourceFilter:
      labelSelector:
        matchLabels:
          env: test
    ttl: 240h
  schedule: 0 0 * * *
status:
  backupDetails:
  - backupStatusInCluster: {}
    clusterKind: AttachedCluster
    clusterName: kurator-member2
```

Analyzing the provided output, let's dissect its sections for a clearer comprehension:

- **Cron Expression in `spec`**:
    - The `schedule` field within the `spec` uses a cron expression. This defines when a backup is to be performed.
    - Once set, the backup won't be executed immediately. Instead, it waits until the time specified by the cron expression.

- **Destination in `spec`**:
    - The `destination` field under `spec` points to `kurator-member2`. This means the backup is specifically for this cluster within its fleet.

- **Policy in `spec`**:
    - The `policy` section outlines the backup strategy. In this instance, it's set to backup resources that match certain labels.
    - For more advanced filtering options, refer to the [Fleet API](https://kurator.dev/docs/references/fleet-api/#fleet)
  
- **Status Section**:
    - The `status` section provides an overview of the backup status across clusters.
    - At the moment, it's empty. As backups are executed according to the cron schedule, this section will populate with relevant details.

### Cleanup

To remove the backup examples used for testing, execute:

```console
kubectl delete backups.backup.kurator.dev specific-ns schedule-matchlabels
```

> Please note: This command only deletes the current object in the k8s API.
For data security, deletion of the data in object storage should be performed using the tools provided by the object storage solution you adopted. 
Kurator currently does not offer capabilities to delete data inside the object storage.
