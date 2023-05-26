---
title: "Integrating the AttachedClusters with Fleet"
linkTitle: "Integrating the AttachedClusters with Fleet"
weight: 15
description: >
  Your roadmap to integrating and managing AttachedClusters with Kurator Fleet.
---

In Kurator, clusters that were not created by Kurator are referred to as `AttachedClusters`.
These clusters can be managed by adding them to the Kurator Fleet, expanding the fleet's control to clusters not originally created by Kurator.

This guide will walk you through the process of creating AttachedCluster resources, using two Kind clusters as examples.

## Prerequisites

### Cluster Operator and kind Cluster

As the AttachedCluster object is controlled by the cluster-operator, you need to first go to [install cluser operator](/docs/setup/install-cluster-operator.md) page to create kind clusters using `hack/local-dev-setup.sh` and install cluster operator.

### AttachedCluster Secrets

From these clusters created by `hack/local-dev-setup.sh`, we'll select kurator-member1 and kurator-member2 to be attached to the Kurator Fleet.

Next, we'll create secrets containing the kubeconfig information of these clusters. Make sure to replace the paths like `/root/.kube/kurator-member1.config` with the actual kubeconfig file paths on your system.

```console
kubectl create secret generic kurator-member1 --from-file=kurator-member1.config=/root/.kube/kurator-member1.config
kubectl create secret generic kurator-member2 --from-file=kurator-member2.config=/root/.kube/kurator-member2.config
```

Please note, here we have named the secrets as `kurator-member1` and `kurator-member2` respectively, and set the key to save the kubeconfig in the secret as `kurator-member1.config` and `kurator-member1.config` respectively.
You can modify these two elements according to your needs.

## Creating AttachedCluster Resources

Now that we have the prerequisites sorted out, let's move on to creating the AttachedCluster resources.

We'll start by editing the configuration for the AttachedCluster.

Notice that the `name` and `key` here need to be consistent with the secret generated earlier.

Edit attachedCluster config.

```console
$ vi ./ac-member1.yaml
apiVersion: cluster.kurator.dev/v1alpha1
kind: AttachedCluster
metadata:
  name: kurator-member1
  namespace: default
spec:
  kubeconfig:
    name: kurator-member1
    key: kurator-member1.config

$ vi ./ac-member2.yaml
apiVersion: cluster.kurator.dev/v1alpha1
kind: AttachedCluster
metadata:
  name: kurator-member2
  namespace: default
spec:
  kubeconfig:
    name: kurator-member2
    key: kurator-member2.config
```

With the configuration ready, we can apply the resources:

```console
kubectl apply -f ./ac-member1.yaml
kubectl apply -f ./ac-member2.yaml
```

## View resource status

Here is an example.

```console
$ kubectl get attachedclusters.cluster.kurator.dev kurator-member1 -o yaml
kind: AttachedCluster
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"cluster.kurator.dev/v1alpha1","kind":"AttachedCluster","metadata":{"annotations":{},"name":"kurator-member1","namespace":"default"},"spec":{"kubeconfig":{"key":"kubeconfig","name":"kurator-member1"}}}
  creationTimestamp: "2023-05-27T09:41:36Z"
  generation: 1
  name: kurator-member1
  namespace: default
  resourceVersion: "28742"
  uid: 46199ce7-3829-4e0a-b1f7-46b47b8d421c
spec:
  kubeconfig:
    key: kubeconfig
    name: kurator-member1
status:
  ready: true
```

## join with fleet

To join the AttachedClusters into a fleet, create the yaml like this:

```console
$ vi ./quickstart.yaml
apiVersion: fleet.kurator.dev/v1alpha1
kind: Fleet 
metadata:
  name: quickstart
  namespace: default
spec:
  clusters:
    # add your AttachedCluster here
    - name: kurator-member1 
      kind: AttachedCluster
    - name: kurator-member2
      kind: AttachedCluster
```

## Cleanup

If you no longer need the AttachedClusters, you can delete them by running the following commands in the terminal:

```console
kubectl delete attachedclusters.cluster.kurator.dev kurator-member1
kubectl delete attachedclusters.cluster.kurator.dev kurator-member2
```
