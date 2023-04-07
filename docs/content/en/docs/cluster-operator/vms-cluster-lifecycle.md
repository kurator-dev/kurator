---
title: "Lifecycle management of cluster on VMs"
linkTitle: "Lifecycle management of cluster on VMs"
weight: 20
description: >
  The easiest way to deploy a cluster on VMs, and manage the cluster lifecycle with Kurator.
---

You can easily manage the VMs cluster with Kurator, including the installation, deletion, upgrade and scale of the VMs cluster.

These properties are built on [Cluster API](https://cluster-api.sigs.k8s.io) and [KubeSpray](https://kubespray.io/).

This guide will describe how to manage the k8s cluster on VMs with Kurator.

## Prerequisites

### Install an SSH key on your VMs

Assuming the public IP address of the VMs where you want to install K8s is "200.x.x.1" and "200.x.x.2".
The private IP address is "192.x.x.1" and "192.x.x.2".

#### Generate a public and private key pair

You can generate a public and private key pair as follows.

```console
ssh-keygen
```

You need follow prompts to "Enter file in which to save the key" and "Enter passphrase".

#### Install SSH public key

```console
ssh-copy-id [user@]200.x.x.1
ssh-copy-id [user@]200.x.x.2
```

#### Check your access

Try logging into the VMs with a password for each login.

```console
ssh 200.x.x.1 
ssh 200.x.x.2
```

### Create the secret for operator

Now you can easily log in VMs, but you still need to pass this ability to Kurator operator through secret.

#### Create the secret with kubectl

Create a secret used to install kubernetes cluster on your VMs via ssh.

```console
kubectl create secret generic cluster-secret --from-file=ssh-privatekey=/root/.ssh/id_rsa
```


#### Check the secret

You can check your secret with follow command.

```console
$ kubectl describe secrets cluster-secret
Name:         cluster-secret
Namespace:    default
Labels:       <none>
Annotations:  <none>

Type:  Opaque

Data
====
ssh-privatekey:  2590 bytes
```

### Customize your cluster configuration

You can find custom cluster examples [here](https://github.com/kurator-dev/kurator/tree/main/examples/infra/customcluster).

Here are the four types of resources needed for VMs cluster provision：

- cluster
- kcp(KubeadmControlPlane)
- customCluster
- customMachine

You can copy the examples and update the parameters as you need:

```console
cp -rfp examples/infra/customcluster examples/infra/my-customcluster
```

First you need to update the host configuration.

```console
$ vi examples/infra/my-customcluster/cc-custommachine.yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: CustomMachine
metadata:
  name: cc-custommachine
  namespace: default
spec:
  master:
    - hostName: master1
      # edit to your real parameters
      publicIP: 200.x.x.1 
      # edit to your real parameters
      privateIP: 192.x.x.1 
      sshKey:
        apiVersion: v1
        kind: Secret
        name: cluster-secret
  node:
    - hostName: node1
      # edit to your real parameters
      publicIP: 200.x.x.2
      # edit to your real parameters
      privateIP: 192.x.x.2
      sshKey:
        apiVersion: v1
        kind: Secret
        name: cluster-secret
```

If you just modify the IP and apply the manifest directly, you will get a kubernetes cluster with one master and one node.

Here are some optional parameters you may care about and the setting position:

| parameters | setting-position                            |
| :-----|:--------------------------------------------| 
| KubeVersion | KubeadmControlPlane.Spec.Version            | 
| PodCIDR | Cluster.Spec.ClusterNetwork.Pods.CidrBlocks | 
| CNIPlugin | CustomCluster.CNI.Type                      | 

## Deploy a k8s cluster on VMs

Now everything ready and let's start to deploy a k8s cluster on your VMs.

### Apply resource configuration

```console
kubectl apply -f examples/infra/my-customcluster
```

### Check your Installation

If you want to see cluster operator log details, you can use following command.

```console
kubectl logs -n kurator-system -l app=kurator-cluster-operator
```

Check the pod installing cluster.

```console
$ kubectl get pod | grep cc-customcluster-init
cc-customcluster-init   0/1     ContainerCreating   0          15s
```

The duration of "ContainerCreating" and "Running" may last tens of minutes which heavily depend on you bandwidth.

Image pull only needs to be executed once in the same cluster.

If you want see the procedures of init worker, you can use the following command.

```console
kubectl logs cc-customcluster-init
```

### Confirm your Installation

When the installation of cluster is done, the status of the init worker will change from "running" to "complete". The phase of customCluster will also change into "succeeded".

You can log in the master node and confirm your installation. Here is an example using cilium as CNI plugin.

```console
$ kubectl get po -A
NAMESPACE     NAME                               READY   STATUS    RESTARTS   AGE
kube-system   cilium-6sjhd                       1/1     Running   0          13m
kube-system   cilium-gsb2g                       1/1     Running   0          13m
kube-system   cilium-operator-57bb669bf6-8nnxp   1/1     Running   0          13m
kube-system   cilium-operator-57bb669bf6-gjbkn   1/1     Running   0          13m
kube-system   coredns-59d6b54d97-gjc56           1/1     Running   0          8m22s
kube-system   coredns-59d6b54d97-ljnfs           1/1     Running   0          2m24s
kube-system   dns-autoscaler-78676459f6-ddkjh    1/1     Running   0          8m20s
kube-system   kube-apiserver-master1             1/1     Running   1          15m
kube-system   kube-controller-manager-master1    1/1     Running   1          14m
kube-system   kube-proxy-5mgc5                   1/1     Running   0          14m
kube-system   kube-proxy-tvjkb                   1/1     Running   0          14m
kube-system   kube-scheduler-master1             1/1     Running   1          14m
kube-system   nginx-proxy-node1                  1/1     Running   0          14m
kube-system   nodelocaldns-97kg7                 1/1     Running   0          8m19s
kube-system   nodelocaldns-fpfxj                 1/1     Running   0          8m19s
```

We can see that the cluster on VMs is installed successful.

## HA for control plane

The cluster installed by Kurator based on kubespray includes a [pre-installed local nginx](https://github.com/kubernetes-sigs/kubespray/blob/master/docs/ha-mode.md) on every non-master Kubernetes node.

If you **don't** want to use pre-installed local nginx and hope to achieve better HA effects, you can also choose to use Kurator to create a cluster bounded with VIP (virtual IPAddress).

In this mode, Kurator utilizes the capabilities of [kube-vip](https://github.com/kube-vip/kube-vip) to enable load-balancing of incoming traffic across multiple control-plane replicas using VIP.

With Kurator, you only need to add a few additional variables in the CRD, then you will get to a high-availability cluster based on kube-vip. The remaining part of this section will explain how to achieve this.


Before proceeding, make sure that you have multiple control plane nodes and have configured them in examples/infra/my-customcluster/cc-custommachine.yaml.

Then, declare the VIP configuration in the customcluster.yaml file. You can edit your configuration as follows.

```console
$ vi examples/infra/my-customcluster/cc-customcluster.yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: CustomCluster
metadata:
  name: cc-customcluster
  namespace: default
spec:
  cni:
    type: cilium
  # add config
  controlPlaneConfig:
    # address is your VIP, assume your VIP is 192.x.x.0
    address: 192.x.x.0
    # optional, sets extra Subject Alternative Names for the API Server signing cert. 
    # If you don't have any want to add, you can directly remove this field.
    certSANs: [200.x.x.1,200.x.x.2]
  machineRef:
    apiVersion: cluster.kurator.dev/v1alpha1
    kind: CustomMachine
    name: cc-custommachine
    namespace: default
```

After editing the cc-customcluster.yaml file, you can apply the configuration by executing the following command, just like above cluster deploying:

```console
$ kubectl apply -f examples/infra/my-customcluster/
```

To confirm your kube-vip installation, you can log into one of the master nodes and view the kube-vip initialization by running the following command:

```console
$ kubectl get po -A | grep kube-vip
kube-system   kube-vip-master1                  1/1     Running   0               13m
kube-system   kube-vip-master2                  1/1     Running   0               9m51s
kube-system   kube-vip-master3                  1/1     Running   0               9m6s
```


## Cluster Scaling

With Kurator, you can declaratively add, remove, or replace multiple worker nodes on VMs.

When performing scaling, you should avoid modifying the hostname in case the same VM has multiple names configured.

You only need to declare the desired final worker node state on the target customMachine, and Kurator can complete the node scaling without any external intervention.

You can make a copy of the custommachine.yaml file and edit it to reflect the desired scaling state.

```console
$ cp examples/infra/my-customcluster/cc-custommachine.yaml examples/infra/my-customcluster/scale.yaml
$ vi examples/infra/my-customcluster/scale.yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: CustomMachine
metadata:
  name: cc-custommachine
  namespace: default
spec:
  master:
    - hostName: master1
      publicIP: 200.x.x.1 
      privateIP: 192.x.x.1 
      sshKey:
        apiVersion: v1
        kind: Secret
        name: cluster-secret
  node:
    # remove node1 for scaling down
    #- hostName: node1
    #  publicIP: 200.x.x.2
    #  privateIP: 192.x.x.2
    #  sshKey:
    #    apiVersion: v1
    #    kind: Secret
    #    name: cluster-secret
    # add node2, node3 for scaling up
    - hostName: node2
      publicIP: 200.x.x.3
      privateIP: 192.x.x.3
      sshKey:
        apiVersion: v1
        kind: Secret
        name: cluster-secret
    - hostName: node3
      publicIP: 200.x.x.4
      privateIP: 192.x.x.4
      sshKey:
        apiVersion: v1
        kind: Secret
        name: cluster-secret
```

Edit the desired state of the new worker nodes in the 'node' field located in the 'spec' section, following the configuration example provided above.

After this, reapply the declaration.

```console
$ kubectl apply -f examples/infra/my-customcluster/scale.yaml
custommachine.infrastructure.cluster.x-k8s.io/cc-custommachine configured
```

Kurator will start working at this point.

Here are some examples of scaling:

### Scaling up

Kurator will compare the provisioned actual cluster state with the current custommachine state, determine the need for scaling up, and automatically execute the scale-up pod.

You can view the running of pods through the following methods:

```console
$ kubectl get pod -A | grep -i scale-up
default              cc-customcluster-scale-up                               1/1     Running     0          103s
```

### Scaling down

Similarly, if deletion is required to achieve the desired state, Kurator will create a pod to remove the worker nodes on VMs.

You can view the running of pods through the following methods:

```console
$ kubectl get pod -A | grep -i scale-down
default              cc-customcluster-scale-down                           1/1     Running     0          37s
```


### Replacing the worker nodes

If the desired state includes both adding and deleting nodes, Kurator will automatically create the pod for adding nodes first, wait for it to complete, and then automatically create the pod for deleting nodes, ultimately achieving the desired state.

You can view the running of pods through the following methods:

```console
$ kubectl get pod -A | grep -i scale-
default              cc-customcluster-scale-up                           1/1     Completed   0          14m
default              cc-customcluster-scale-down                         1/1     Running     0          37s
```

## Cluster upgrading

With Kurator, you can easily upgrade the Kubernetes version of your cluster with a declarative approach.

All you need to do is declaring the desired Kubernetes version on the kcp, and Kurator will complete the cluster upgrade without any external intervention.

Since the upgrade implementation depends on kubeadm, it is recommended to avoid skipping minor versions. For example, you can upgrade from v1.22.0 to v1.23.9, but you **cannot** upgrade from v1.22.0 to v1.24.0 in one step.

To declare the desired upgrading version, you can just edit the CRD of kcp to reflect the desired upgrading version:

```console
# you may need replace "cc-kcp" to your kcp crd
$ kubectl edit kcp cc-kcp 
  ...
spec:
  kubeadmConfigSpec:
    format: cloud-config
  machineTemplate:
    infrastructureRef:
      apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
      kind: customMachine
      name: cc-custommachine
      namespace: default
    metadata: {}
  replicas: 1
  rolloutStrategy:
    rollingUpdate:
      maxSurge: 1
    type: RollingUpdate
  # edit the version to desired upgrading version
  version: v1.24.6
status:
  conditions:
  ...
```

To confirm the upgrade worker is running, use the following command:

```console
$ kubectl get pod -A | grep -i upgrade
default              cc-customcluster-upgrade                                1/1     Running     0               18s
```

## Delete the k8s cluster for VMs

If you no longer need cluster on VMs and want to delete the cluster, you only need to delete the cluster object.

### Find the cluster resource

Check the cluster object you have.

```console
$ kubectl get clusters.cluster.x-k8s.io
NAME         PHASE      AGE    VERSION
cc-cluster   running    3h6m   
```

### Delete the custom cluster resource

Delete the custom cluster which you want.

```console
kubectl delete cluster cc-cluster 
```

The deleting action of cluster will get stuck due to the procedures of deletion of related object.

The deletion will create a terminating worker. The terminating worker will be responsible for cleaning up the cluster on the VMs.

You can open a new command tab in terminal to check the status of the terminate pod.

```console
$ kubectl get pod | grep terminate
cc-customcluster-terminate   1/1     Running   0          14s
```

After the terminate worker finished, the Cluster on VMs will be clean up and the related resource will also be deleted.

The entire deletion of cluster may take 5 min.
