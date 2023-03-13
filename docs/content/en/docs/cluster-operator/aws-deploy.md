---
title: "Deploy Cluster on AWS"
linkTitle: "Deploy Cluster on AWS"
weight: 30
description: >
  The easiest way to deploy cluster on AWS with Kurator.
---

In this tutorial we’ll cover the basics of how to use [Cluster API](https://cluster-api.sigs.k8s.io) and kurator cluster operator to create kubernetes cluster.

## Prepare AWS Credentials

{{< boilerplate prepare-aws >}}

## Create a vanilla cluster on AWS

The clusterctl CLI tool handles the lifecycle of a Cluster API managed cluster.

```console
# download clusterctl
curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.2.5/clusterctl-linux-amd64 -o clusterctl
sudo install -o root -g root -m 0755 clusterctl /usr/local/bin/clusterctl
# verify version
clusterctl version
```

Create a kubernetes cluster on AWS, which contains control plane nodes and worker nodes, the cluster topology shows as follows:

{{< image width="75%"
    link="./image/capa-aws.png"
    >}}

Generate the cluster configuration:

```console
export AWS_CONTROL_PLANE_MACHINE_TYPE=t3.large
export AWS_NODE_MACHINE_TYPE=t3.large
export AWS_REGION=us-east-1
export AWS_SSH_KEY_NAME=default
export KUBERNETES_VERSION=v1.25.0
export CONTROL_PLANE_MACHINE_COUNT=3
export WORKER_MACHINE_COUNT=3
clusterctl generate cluster capi-quickstart --infrastructure=aws:v2.0.0 > examples/infra/capi-quickstart.yaml
```

The cluster resource topology shows as follows:

{{< image width="75%"
    link="./image/capa-crd.png"
    >}}


Apply the cluster manifest:

```console
kubectl apply -f examples/infra/capi-quickstart.yaml
```

> if you want create a cluster with multi instance types, please checkout the [multi nodes demo](https://github.com/kurator-dev/kurator/blob/main/examples/infra/multi-tenancy/capi-nodes.yaml)

Wait the control plane is up:

```console
kubectl get kubeadmcontrolplane -w
```

***The control plane won’t be Ready until we install a CNI in the next step.***

```console
# retrieve the cluster Kubeconfig 
clusterctl get kubeconfig capi-quickstart > /root/.kube/capi-quickstart.kubeconfig
# deploy calico solution
kubectl --kubeconfig=/root/.kube/capi-quickstart.kubeconfig apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.24.1/manifests/calico.yaml
# After a short while, our nodes should be running and in Ready state
kubectl --kubeconfig=/root/.kube/capi-quickstart.kubeconfig get nodes
```

## Cleanup

{{< boilerplate cleanup >}}
