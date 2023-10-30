---
title: "Install Backup Plugin"
linkTitle: "Install Backup Plugin"
weight: 10
description: >
  Configure backup plugin for Fleet to enable Kurator's capabilities.
---

To support Kurator's unified backup, restore, and migration features, it's imperative to first configure the backup plugin for [Fleet](https://kurator.dev/docs/references/fleet-api/#fleet). This guide will walk you through configuring the [Velero](https://velero.io/)-based backup plugin for Fleet, laying the groundwork for Kurator's unified backup, restore, and migration capabilities.

## Prerequisites

### 1. Fleet Manager Setup

Set up the Fleet manager by following the instructions in the [installation guide](/docs/setup/install-fleet-manager/).

### 2. Configuring Object Storage for Velero

Kurator's backup relies on Velero. Hence, [Object Storage](https://velero.io/docs/v1.12/how-velero-works/) is essential to store Kubernetes resources. We support various object storage options. In this guide, we provide detailed usage instructions for the following configurations:

- [Minio](https://min.io/): Install a Minio service in the current Kurator host cluster for local validation. Refer to the [installation guide](/docs/setup/install-minio).

- Cloud Service Provider's Object Storage (using Huawei Cloud OBS as an example): Purchase or use an existing object storage service.

> **Note**: The Minio method is intended only for validation purposes. For production environments, it's recommended to use cloud service providers' storage services.

After choosing the storage method, create a secret with the access details. Use this secret's name as a parameter for the fleet.

For Minio: [Minio installation guide](/docs/setup/install-minio).

For cloud storage using AK and SK (e.g., OBS):

```console
kubectl create secret generic obs-credentials --from-literal=access-key={YOUR_ACCESS_KEY} --from-literal=secret-key={YOUR_SECRET_KEY}
```

For more configuration options, please refer to the [Fleet API](https://kurator.dev/docs/references/fleet-api/).

### 3. Secrets and Setup for Attached Clusters

```console
kubectl create secret generic kurator-member1 --from-file=kurator-member1.config=/root/.kube/kurator-member1.config
kubectl create secret generic kurator-member2 --from-file=kurator-member2.config=/root/.kube/kurator-member2.config
kubectl apply -f - <<EOF
apiVersion: cluster.kurator.dev/v1alpha1
kind: AttachedCluster
metadata:
  name: kurator-member1
  namespace: default
spec:
  kubeconfig:
    name: kurator-member1
    key: kurator-member1.config
---
apiVersion: cluster.kurator.dev/v1alpha1
kind: AttachedCluster
metadata:
  name: kurator-member2
  namespace: default
spec:
  kubeconfig:
    name: kurator-member2
    key: kurator-member2.config
EOF
```

## Create a Fleet with the Backup Plugin Enabled

### Use Minio

If you opt to use Minio, apply the following configuration:

```console
kubectl apply -f - <<EOF
apiVersion: fleet.kurator.dev/v1alpha1
kind: Fleet
metadata:
  name: quickstart
  namespace: default
spec:
  clusters:
    - name: kurator-member1
      kind: AttachedCluster
    - name: kurator-member2
      kind: AttachedCluster
  plugin:
    backup:
      storage:
        location:
          bucket: velero
          provider: aws
          endpoint: http://172.18.255.200:9000
          region: minio
        secretName: minio-credentials
EOF
```

> **Note**: The `endpoint`(`http://172.18.255.200:9000`) in `fleet-minio.yaml` is depends on your minio service ip, check your minio service ip in [Minio installation guide](/docs/setup/install-minio).

### Use OBS

If you choose to use OBS, apply the following configuration:

```console
kubectl apply -f - <<EOF
apiVersion: fleet.kurator.dev/v1alpha1
kind: Fleet
metadata:
  name: quickstart
  namespace: default
spec:
  clusters:
    - name: kurator-member1
      kind: AttachedCluster
    - name: kurator-member2
      kind: AttachedCluster
  plugin:
    backup:
      storage:
        location:
          bucket: kurator-obs
          provider: huaweicloud
          endpoint: http://obs.cn-south-1.myhuaweicloud.com
          region: cn-south-1
        secretName: obs-credentials
EOF
```

### Fleet Backup Plugin Configuration Explained

Let's delve into the `spec` section of the above Fleet:

- `clusters`: Contains the two `AttachedCluster` objects created earlier, indicating that the backup plugin will be installed on these two clusters.
  
- `plugin`: The `backup` indicates the description of a backup plugin. Currently, it only contains storage related configurations. For more configuration options, please refer to the [Fleet API](https://kurator.dev/docs/references/fleet-api/). 

   Within this storage configuration, users should ensure that the details in the location field are accurate. This information primarily differentiates the configurations in fleet-obs.yaml from fleet-minio.yaml. Additionally, the secretName refers to the name of the secret that was established earlier within the same namespace.

## Verify the Installation

To ensure that the backup plugin is successfully installed and running, follow the steps below:

### 1. Check Velero Pods

Run the following commands:

```console
kubectl get pod -n velero --kubeconfig=/root/.kube/kurator-member1.config
kubectl get pod -n velero --kubeconfig=/root/.kube/kurator-member2.config
```

Initially, you should observe:

```plaintext
velero               velero-velero-kurator-member1-upgrade-crds-bm28q        0/1     Init:0/1   0          31s
velero               velero-velero-kurator-member2-upgrade-crds-lg7gd        0/1     Init:0/1   0          28s
```

After waiting for about 2 or more minutes, check again to ensure all pods are in the `Running` state:

```plaintext
velero               node-agent-hn7h5                                        1/1     Running   0          85s
velero               velero-velero-kurator-member1-755d5675ff-sbrkg          1/1     Running   0          85s
velero               node-agent-2mrnj                                        1/1     Running   0          116s
velero               velero-velero-kurator-member2-c5b87598b-4zfsc           1/1     Running   0          116s
```

### 2. Confirm Connection to the Object Storage

Run the following commands to ensure backup has successfully connected to the object storage:

```console
kubectl get backupstoragelocations.velero.io -A --kubeconfig=/root/.kube/kurator-member1.config
kubectl get backupstoragelocations.velero.io -A --kubeconfig=/root/.kube/kurator-member2.config
```

When the `PHASE` for all is `Available`, it indicates that the object storage is accessible:

```plaintext
NAMESPACE   NAME      PHASE       LAST VALIDATED   AGE     DEFAULT
velero      default   Available   57s              8m23s   true
```

## Cleanup

This section guides you through the process of cleaning up the fleets and plugins.

### 1. Cleanup the Backup Plugin

If you only need to remove the backup plugin, simply edit the current fleet and remove the corresponding description:

```console
kubectl edit fleet.fleet.kurator.dev quickstart
```

To check the results of the deletion, you can observe that the Velero components have been removed:

```console
kubectl get po -A --kubeconfig=/root/.kube/kurator-member1.config
kubectl get po -A --kubeconfig=/root/.kube/kurator-member2.config
```

If you wish to reinstall the components later, you can simply edit the fleet and add the necessary configurations.

### 2. Cleanup the Fleet

When the fleet is deleted, all associated plugins will also be removed:

```console
kubectl delete fleet.fleet.kurator.dev quickstart
```
