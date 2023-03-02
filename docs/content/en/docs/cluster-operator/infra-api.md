---
title: "Deploy Cluster with Kurator Cluster API"
linkTitle: "Deploy Cluster with Kurator Cluster API"
description: >
  The easiest way to deploy cluster with Kurator Cluster API.
---

In this tutorial we’ll cover the basics of how to use [Cluster API](https://github.com/kurator-dev/kurator/blob/main/pkg/apis/cluster/v1alpha1/cluster_types.go) to provision Kubernetes clusters.

## Prerequisites

{{% readfile "includes/prerequisites.tpl" %}}

## Build `cluster-operator` from source

{{% readfile "includes/build-from-source.tpl" %}}

## Install cluster operator

***Please make sure cert manager is ready before install cluster operator

{{% readfile "includes/install-cluster-operator.tpl" %}}

## Create a vanilla cluster with Infra API

Apply the cluster manifest:

```console
kubectl apply -f examples/cluster/quickstart.yaml
```

Wait the control plane is up:

```console
kubectl get cluster -w
```

Retrieve the cluster's Kubeconfig:

```console
clusterctl get kubeconfig quickstart > /root/.kube/quickstart.kubeconfig
```

Check node state:

```console
kubectl --kubeconfig=/root/.kube/quickstart.kubeconfig get nodes
```

## Cleanup

{{% readfile "includes/cleanup.tpl" %}}
