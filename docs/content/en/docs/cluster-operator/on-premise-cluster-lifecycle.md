---
title: "On-Premise Kubernetes Cluster Lifecycle Management "
linkTitle: "On-Premise Kubernetes Cluster Lifecycle Management"
weight: 20
description: >
  The easiest way to deploy an on-premise Kubernetes cluster and manage its lifecycle using Kurator.
---

This guide offers the simplest method to deploy an on-premise Kubernetes cluster and manage its lifecycle using Kurator. 
Kurator allows you to manage your on-premise cluster effortlessly, including the installation, deletion, upgrade, and scaling of the cluster.

These properties are built on [Cluster API](https://cluster-api.sigs.k8s.io) and [KubeSpray](https://kubespray.io/).

## Prerequisites

### Install an SSH key on your on-premise servers

Assume the public IP addresses of the servers where you plan to install Kubernetes are "200.x.x.1" and "200.x.x.2". 
The corresponding private IP addresses are "192.x.x.1" and "192.x.x.2".

#### Generate a public and private key pair

You can generate a public and private key pair as follows.

```console
ssh-keygen
```

You need follow prompts to "Enter file in which to save the key" and "Enter passphrase".

#### Install SSH public key

Attempt to log into the servers using a password for each login:

```console
ssh-copy-id 200.x.x.1
ssh-copy-id 200.x.x.2
```

#### Verify your access

Try logging into the on-premise service with a password for each login.

```console
ssh 200.x.x.1 
ssh 200.x.x.2
```

### Create the secret for operator

Although you can easily log into your servers now, you still need to grant this ability to the Kurator operator through a secret.

#### Create the secret with kubectl

Create a secret used to install Kubernetes cluster on your servers via SSH:

```console
kubectl create secret generic cluster-secret --from-file=ssh-privatekey=/root/.ssh/id_rsa
```


#### Verify the secret

You can verify your secret with the following command:

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

Here are the four types of resources needed for on-premise Kubernetes cluster provision：

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

If you just modify the IP and apply the manifest directly, you will get a Kubernetes  cluster with one master and one node.

Here are some optional parameters that you may be interested in, along with their settings:

| Parameters     | Setting Position                                                                |
|:--------------|:----------------------------------------------------------------| 
| KubeVersion   | KubeadmControlPlane.Spec.Version                                | 
| PodCIDR       | Cluster.Spec.ClusterNetwork.Pods.CidrBlocks                     | 
| ServiceCIDR   | Cluster.Spec.ClusterNetwork.Services.CidrBlocks                 |
| CNIPlugin     | CustomCluster.CNI.Type                                          | 
| KubeImageRepo | kcp.Spec.KubeadmConfigSpec.ClusterConfiguration.ImageRepository |

## Deploy an On-Premise Kubernetes Cluster

Now everything is ready. Let's begin deploying a Kubernetes cluster on your on-premise servers:

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

If you want to see the procedures of the init worker, you can use the following command:

```console
kubectl logs cc-customcluster-init
```

### Verify your Kubernetes Cluster

When the installation of cluster is done, the status of the init worker will change from "running" to "complete". The phase of customCluster will also change into "succeeded".

You can log into the master node and verify your Kubernetes cluster. Here is an example master node using Cilium as the CNI plugin:

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

You should see that the on-premise cluster has been successfully installed.

## High Availability for the Control Plane

The cluster installed by Kurator, based on KubeSpray, includes a [pre-installed local nginx](https://github.com/kubernetes-sigs/kubespray/blob/master/docs/ha-mode.md) on every non-master Kubernetes node.

If you prefer not to use the pre-installed local Nginx and aim to achieve better high-availability (HA) effects, you can opt to use Kurator to create a cluster bound with a Virtual IP (VIP).

In this mode, Kurator utilizes the capabilities of [kube-vip](https://github.com/kube-vip/kube-vip) to enable load-balancing of incoming traffic across multiple control-plane replicas using VIP.

With Kurator, you only need to add a few additional variables in the CRD, then you will get a high-availability cluster based on kube-vip after init worker finished. The remaining part of this section will explain how to achieve this.

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
    # loadBalancerDomainName is an optional field that sets the load balancer domain name. 
    # If not specified, the default name, controlPlaneConfig.address is used. 
    loadBalancerDomainName: my-apiserver-lb.kurator.com
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
kubectl apply -f examples/infra/my-customcluster/
```

To confirm your kube-vip installation, you can log into one of the master nodes and view the kube-vip initialization by running the following command:

```console
$ kubectl get po -A | grep kube-vip
kube-system   kube-vip-master1                  1/1     Running   0               13m
kube-system   kube-vip-master2                  1/1     Running   0               9m51s
kube-system   kube-vip-master3                  1/1     Running   0               9m6s
```


## Cluster Scaling

With Kurator, you can declaratively add, remove, or replace multiple worker nodes on on-premise servers.

When performing scaling, avoid modifying the hostname in case the same server has multiple names configured.

Declare the desired final worker node state on the target customMachine, and Kurator completes the node scaling without any external intervention.

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

Similarly, if deletion is required to achieve the desired state, Kurator will create a pod to remove a number of worker nodes.

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

Declare the desired Kubernetes version on the kcp, and Kurator completes the cluster upgrade without any external intervention.

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
  version: v1.26.5
status:
  conditions:
  ...
```

Confirm the upgrade worker is running with the following command:

```console
$ kubectl get pod -A | grep -i upgrade
default              cc-customcluster-upgrade                                1/1     Running     0               18s
```

## Delete the k8s cluster for on-premise servers

If you no longer need the cluster on on-premise servers and want to delete the cluster, just delete the cluster object.

### Find the cluster resource

Check the cluster object you have.

```console
$ kubectl get clusters.cluster.x-k8s.io
NAME         PHASE      AGE    VERSION
cc-cluster   running    3h6m   
```

### Delete the custom cluster resource

Delete the custom cluster that you want to remove.

```console
kubectl delete clusters.cluster.x-k8s.io cc-cluster
```

The deleting action of the cluster might get stuck due to the procedures of deletion of related object.

The deletion will create a terminating worker. The terminating worker will be responsible for cleaning up the cluster on the on-premise servers.

You can open a new command tab in terminal to check the status of the terminate pod.

```console
$ kubectl get pod | grep terminate
cc-customcluster-terminate   1/1     Running   0          14s
```

After the terminate worker finishes, the cluster on on-premise servers will be cleaned up and the related resource will also be deleted.

The entire deletion of cluster may take 5 min.
