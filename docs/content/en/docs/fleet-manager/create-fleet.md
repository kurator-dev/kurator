---
title: "Get Started with Kurator Fleet"
linkTitle: "Get Started with Kurator Fleet"
weight: 10
description: >
  The easiest way to manage multi clusters with fleet manager.
---

In this tutorial we’ll cover the basics of how to use [Fleet](https://kurator.dev/docs/references/fleet-api/#fleet) to manage a group of clusters.

## Prerequisites

Now fleet manager can  manage clusters either created using kurator [Cluster API](https://kurator.dev/docs/references/cluster-api/#cluster.kurator.dev/v1alpha1.Cluster) or prebuilt [AttachedCluster](https://kurator.dev/docs/references/cluster-api/#cluster.kurator.dev/v1alpha1.AttachedCluster). 

Please refer to [Get started with Kurator Cluster API](/docs/cluster-operator/kurator-cluster-api) to create a vanilla kubernetes cluster.

## Create a Fleet

Here in this example, we create a fleet with a cluster provisioned with kurator [cluster operator](/docs/cluster-operator). If you want to manage a prebuilt cluster, please refer to [Manage AttachedCluster](/docs/fleet-manager/manage-attachedcluster).


### Apply the fleet manifest

```console
kubectl apply -f examples/fleet/fleet.yaml
```

After a while, we can see the fleet turns ready, the fleet has one cluster `quickstart` registered successfully. 

```console
$ kubectl get fleet quickstart -n test -oyaml
apiVersion: fleet.kurator.dev/v1alpha1
kind: Fleet
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"fleet.kurator.dev/v1alpha1","kind":"Fleet","metadata":{"annotations":{},"name":"quickstart","namespace":"test"},"spec":{"clusters":[{"name":"quickstart"}]}}
  creationTimestamp: "2023-04-10T02:24:12Z"
  finalizers:
  - fleet.kurator.dev
  generation: 1
  name: quickstart
  namespace: test
  resourceVersion: "6317753"
  uid: 5483c4d2-0ccf-48f8-afae-945b18dac8d9
spec:
  clusters:
  - name: quickstart
status:
  credentialSecret: kubeconfig
  phase: Ready
  readyClusters: 1
```

## Cleanup

Delete the fleet created

```console
kubectl delete fleet quickstart
```

Uninstall fleet manager:

```console
helm uninstall kurator-fleet-manager -n kurator-system
```

{{< boilerplate cleanup >}}
