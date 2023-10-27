---
title: "Enable Policy Management with fleet"
linkTitle: "Policy Management"
weight: 30
description: >
  The easiest way to enable policy management with fleet.
---

In this tutorial, we will cover how to implement unified distributed storage on a set of clusters, using [Fleet](https://kurator.dev/docs/references/fleet-api/#fleet)

## Architecture

Fleet's unified distributed storage is built on top of [Rook](https://rook.io/), and the overall architecture is shown below:

{{< image width="100%"
    link="./image/fleet-policy.drawio.svg"
    >}}

## Prerequisites

### 1. Fleet Manager Setup

Set up the Fleet manager by following the instructions in the [installation guide](/docs/setup/install-fleet-manager/).

### 2. Rook prerequisites

1. Kubernetes v1.22 or higher is supported
1. To configure the Ceph storage cluster, at least one of these local storage types is required:

   - Raw devices (no partitions or formatted filesystems)
   - Raw partitions (no formatted filesystem)
   - LVM Logical Volumes (no formatted filesystem)
   - Persistent Volumes available from a storage class in `block` mode

The easiest way to do this is to mount an Raw disk on the nodes.

### 3. Secrets and Setup for Attached Clusters

1. Running the following command to create two secrets to access attached clusters.

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

## Create a Fleet with the DistributedStorage Plugin Enabled

Run following command to create rook operator and rook ceph cluster in the Fleet:

```console
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
  plugin:
    distributedStorage:
      storage:
        dataDirHostPath: /var/lib/rook
        monitor:
          count: 3
          labels:
            role: MonitorNodeLabel
        manager:
          count: 2
          annotations:
            role: ManagerNodeAnnotation

```

### Fleet DistributedStorage Plugin Configuration Explained

Let's delve into the `spec` section of the above Fleet:

- `clusters`: Contains the two `AttachedCluster` objects created earlier, indicating that the distributedstorage plugin will be installed on these two clusters.
- `plugin`: The `distributedStorage` indicates the description of a distributedstorage plugin. For more configuration options, please refer to the [Fleet API](https://kurator.dev/docs/references/fleet-api/).

## Verify the Installation

To ensure that the distributedstorage plugin is successfully installed and running, run the following commands:

```console
kubectl get po -A -n rook-ceph --kubeconfig=/root/.kube/kurator-member1.config
kubectl get po -A -n rook-ceph --kubeconfig=/root/.kube/kurator-member2.config
```

After waiting for some time check the status of all pods under the rook-ceph namespace. The result is shown below:

```console
NAME                                                                READY   STATUS             RESTARTS   AGE
csi-cephfsplugin-75lqn                                              2/2     Running            0          32h
csi-cephfsplugin-bd7d6                                              2/2     Running            0          32h 
csi-cephfsplugin-lk6wz                                              2/2     Running            0          32h  
csi-cephfsplugin-provisioner-86788ff996-2vfg2                       5/5     Running            0          32h
csi-cephfsplugin-provisioner-86788ff996-bc6cg                       5/5     Running            0          32h
csi-rbdplugin-28s5x                                                 2/2     Running            0          32h
csi-rbdplugin-7pj48                                                 2/2     Running            0          32h
csi-rbdplugin-ffpxn                                                 2/2     Running            0          32h
csi-rbdplugin-provisioner-7b5494c7fd-bxm4f                          5/5     Running            0          32h
csi-rbdplugin-provisioner-7b5494c7fd-nm4kp                          5/5     Running            0          32h
rook-ceph-crashcollector-testpool-dev-linux-0001-6cfb5c54ff-mf6l4   1/1     Running            0          32h
rook-ceph-crashcollector-testpool-dev-linux-0003-84f6f85cb7-9grjp   1/1     Running            0          32h
rook-ceph-crashcollector-testpool-dev-linux-0004-65bf4f84c4-cvwpp   1/1     Running            0          32h
rook-ceph-mds-ceph-filesystem-a-65688966df-5zq7d                    2/2     Running            0          32h
rook-ceph-mds-ceph-filesystem-a-6d5dcb85b6-k8pqr                    2/2     Running            0          32h
rook-ceph-mgr-a-8456d8cc98-n54n2                                    3/3     Running            0          32h
rook-ceph-mgr-b-67c9cc8c95-gk82m                                    3/3     Running            0          32h
rook-ceph-mon-a-68659ffbfb-bqn6q                                    2/2     Running            0          32h
rook-ceph-mon-b-85479654f8-n26vs                                    2/2     Running            0          32h
rook-ceph-mon-c-6779986564-p8dmc                                    2/2     Running            0          32h
rook-ceph-operator-b89ccd545-vlg2r                                  1/1     Running            0          32h
rook-ceph-osd-0-996475cb6f-cfbwh                                    1/1     Running            0          32h
rook-ceph-osd-1-67f5ff649c-vs5qt                                    1/1     Running            0          32h
rook-ceph-osd-2-7d4c78b74b-l8jg5                                    1/1     Running            0          32h
rook-ceph-osd-prepare-testpool-dev-linux-0001-jz766                 0/1     Completed          0          32h
rook-ceph-osd-prepare-testpool-dev-linux-0003-t47st                 0/1     Completed          0          32h
rook-ceph-osd-prepare-testpool-dev-linux-0004-kw94n                 0/1     Completed          0          32h
rook-ceph-rgw-ceph-objectstore-a-5c4df48bbb-bf6jn                   2/2     Running            0          32h
```

## Storage Configuration

After rook opeartor and rook ceph cluster are installed, this chapter provides examples of using Block Storage, Filesystem Storage and Object Storage.

### Block Storage Class Configuration

Block storage allows a single pod to mount storage. This section how to create a block storage persistent volume kubernetes pod that uses the block storage provided by Rook.

The `StorageClass` and `CephBlockPool` need to be created before the Rook can configure storage.This will allow Kubernetes to interact with the Rook when configuring persistent volumes.

Run the following commands, can create a `CephBlockPool`:

```console
kubectl apply -f - <<EOF
apiVersion: ceph.rook.io/v1
kind: CephBlockPool
metadata:
  name: replicapool
  namespace: rook-ceph
spec:
  failureDomain: host
  replicated:
    size: 3
EOF
```

Run the following commands, can create a block mod `StorageClass`:

```console
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: rook-ceph-block
provisioner: rook-ceph.rbd.csi.ceph.com
parameters:
    clusterID: rook-ceph
    pool: replicapool
    imageFormat: "2"
    imageFeatures: layering

    csi.storage.k8s.io/provisioner-secret-name: rook-csi-rbd-provisioner
    csi.storage.k8s.io/provisioner-secret-namespace: rook-ceph
    csi.storage.k8s.io/controller-expand-secret-name: rook-csi-rbd-provisioner
    csi.storage.k8s.io/controller-expand-secret-namespace: rook-ceph
    csi.storage.k8s.io/node-stage-secret-name: rook-csi-rbd-node
    csi.storage.k8s.io/node-stage-secret-namespace: rook-ceph
    csi.storage.k8s.io/fstype: ext4

reclaimPolicy: Delete
allowVolumeExpansion: true
```

There are a few things to note in the above block storage class configuration:

- provisioner is configured in the format (operator-namespace).rbd.csi.ceph.com. Change "rook-ceph" provisioner prefix to match the operator namespace if needed.
- `parametes.clusterID` is the namespace where the rook cluster is running.
- `parametes.pool` is the `CephBlockPool` created before.

### FileSystem Storage Class Configuration

A filesystem storage (also named shared filesystem) can be mounted with read/write permission from multiple pods. This may be useful for applications which can be clustered using a shared filesystem.This section describes how to create a kubernetes pod of file storage persistent volumes using the file storage provided by Rook.

Using the file storage provided by Rook, you can set up the metadatapool, datapool, and metadata server, all of which can be set up in the `CephFileSystem`. Run the following commands, can create a `CephFileSystem`:

```console
kubectl apply -f - <<EOF
apiVersion: ceph.rook.io/v1
kind: CephFilesystem
metadata:
  name: ceph-filesystem
  namespace: rook-ceph
spec:
  metadataPool:
    replicated:
      size: 3
  dataPools: 
    - name: data0
      replicated:
        size: 3
  preserveFilesystemOnDelete: true
  metadataServer:
    activeCount: 1
    activeStandby: true
```

The Rook operator will create all the pools and other resources necessary to start the service. To confirm the filesystem is configured, wait for the mds pods to start:

```console
kubectl get po -n rook-ceph -l app=rook-ceph-mds
NAME                                                     READY   STATUS       RESTARTS   AGE
rook-ceph-mds-ceph-filesystem-a-65688966df-5zq7d         2/2     Running      0          32h
rook-ceph-mds-ceph-filesystem-a-6d5dcb85b6-k8pqr         2/2     Running      0          32h
```

Before you can use the file storage provided by Rook, you need to create a storage class for the file storage type, which is required for Kubernetes to interoperate with the CSI driver to create persistent volumes. Apply this storage class definition as:

```console
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: rook-cephfs
provisioner: rook-ceph.cephfs.csi.ceph.com
parameters:
  clusterID: rook-ceph
  fsName: ceph-filesystem
  pool: data0
  csi.storage.k8s.io/provisioner-secret-name: rook-csi-cephfs-provisioner
  csi.storage.k8s.io/provisioner-secret-namespace: rook-ceph
  csi.storage.k8s.io/controller-expand-secret-name: rook-csi-cephfs-provisioner
  csi.storage.k8s.io/controller-expand-secret-namespace: rook-ceph
  csi.storage.k8s.io/node-stage-secret-name: rook-csi-cephfs-node
  csi.storage.k8s.io/node-stage-secret-namespace: rook-ceph
reclaimPolicy: Delete
EOF
```

There are a few things to note in the above filesystem storage class configuration:

- provisioner is configured in the format (operator-namespace).cephfs.csi.ceph.com. Change "rook-ceph" provisioner prefix to match the operator namespace if needed.
- `parameters.fsName` and `parameters.pool` needs to correspond to the name created in `CephFileSystem`.

### Object Storage Class Configuration

In Rook, object storage exposes the S3 API to the storage cluster so that applications in the cluster can store data. As same as file storage, object storage requires metadata pool, data pool, and rgw a specific plug-in to support object storage. All of this can be set up in the `CephObjectStorage`. Run the following commands, can create a `CephObjectStorage`:

```console
kubectl apply -f - <<EOF
apiVersion: ceph.rook.io/v1
kind: CephObjectStore
metadata:
  name: ceph-objectstorage
  namespace: rook-ceph
spec:
  metadataPool:
    failureDomain: host
    replicated:
      size: 3
  dataPool:
    failureDomain: host
    erasureCoded:
      dataChunks: 2
      codingChunks: 1
  preservePoolsOnDelete: true
  gateway:
    sslCertificateRef:
    port: 80
    instances: 1
EOF
```

The Rook operator will create all the pools and other resources necessary to start the service. To confirm the object storage is configured, wait for the rgw pods to start:

```console
kubectl get po -n rook-ceph -l app=rook-ceph-rgw
NAME                                                     READY   STATUS       RESTARTS   AGE
rook-ceph-rgw-ceph-objectstore-a-5c4df48bbb-bf6jn        2/2     Running      0          32h
```

Now that the object store is configured, we next need to create a storage bucket that will allow clients to read and write objects. Buckets can be created by defining storage classes, similar to the pattern used for block and file storage. Apply this storage class definition as:

```console
kubectl apply -f - <<EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: rook-ceph-bucket
provisioner: rook-ceph.ceph.rook.io/bucket
reclaimPolicy: Delete
parameters:
  objectStoreName: ceph-objectstore
  objectStoreNamespace: rook-ceph
EOF
```

There are a few things to note in the above filesystem storage class configuration:

- provisioner is configured in the format (operator-namespace).ceph.rook.io/bucket. Change "rook-ceph" provisioner prefix to match the operator namespace if needed.
- `parameters.objectStorageName` needs to correspond to the name created in `CephObjectStorage`

### Use Block Storage/Filesystem Storage/Object Storage 

After creating the storagec class for block, file and object storage, it's time to actually use this storage class. We need to declare Persistent Volume Claims that use these storage classes to complete the storage. Here is an example:

```console
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata: 
  name: block-pvc
spec:
  storageClassName: rook-ceph-block
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF
```

You can use different types of storage by changing the name of the storage class corresponding to the `spec.storageclassname`.

Reference the created Persistent Volume Claims to the Pod for use. Here is an example of how to use it:

```console
kubectl apply -f -<<EOF
apiVersion: v1
kind: Pod 
metadata:
  name: nginx-blockstorage-test
spec:
  containers:
    - name: app-container
      image: nginx
      volumeMounts:
      - name: block-volume 
        mountPath: "/etc/nginx"
  volumes:
    - name: block-volume
      persistentVolumeClaim:
        claimName: block-pvc
```

The above command creates a nginx-blockstorage-test pod and mounts Persistent Volume in its own Pod. You can run the following command to see the kubernetes volume claims:

```console
kubectl get pvc -A
NAME          STATUS     VOLUME                                       CAPACITY      ACCESSMODES     AGE
block-pvc     Bound      pvc-84dc454c-efc1-11e6-bc9a-0cc47a3459ee     1Gi           RWO             16h
```

## Cleanup

This section guides you through the process of cleaning up the fleets and plugins.

### 1. Cleanup the Backup Plugin

If you only need to remove the distributedstorage plugin, simply edit the current fleet and remove the corresponding description:

```console
kubectl edit fleet.fleet.kurator.dev quickstart
```

To check the results of the deletion, you can observe that the Velero components have been removed:

```console
kubectl get po -A --kubeconfig=/root/.kube/kurator-member1.config
kubectl get po -A --kubeconfig=/root/.kube/kurator-member2.config
```

Perhaps there are still some rook components left over, which can be executed by running the following command:

```console
kubectl api-resources --verbs=list --namespaced -o name | xargs -n 1 kubectl get --show-kind --ignore-not-found -n rook-ceph
```

After getting undeleted rook components, you can delete it by editing its configuration file via `kubectl edit` and removing its finalizer.

More information on removing a Rook can be found at [rook cleanup guide](https://rook.io/docs/rook/v1.12/Storage-Configuration/ceph-teardown/)

If you wish to reinstall the components later, you can simply edit the fleet and add the necessary configurations.

### 2. Cleanup the Fleet

When the fleet is deleted, all associated plugins will also be removed:

```console
kubectl delete fleet.fleet.kurator.dev quickstart
```
