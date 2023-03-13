---
title: "Setup a cluster with IRSA enabled"
linkTitle: "Deploy a cluster with IRSA enabled"
weight: 11
description: >
  Setup a clsuter allows applications in a pod's containers can use an AWS SDK or the AWS CLI to make API requests to AWS services using AWS IAM.
---

In this tutorial we’ll cover the basics of how to use [Cluster API](https://github.com/kurator-dev/kurator/blob/main/pkg/apis/cluster/v1alpha1/cluster_types.go) to provision Kubernetes clusters.

## Setup a cluster on AWS with IRSA enabled

IRSA(IAM roles for service accounts) allows applications in a pod's containers can use an AWS SDK or the AWS CLI to make API requests to AWS services using AWS Identity and Access Management (IAM) permissions. More details can be found [here](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html).

### Setup cluster

Apply the cluster manifest:

```console
kubectl apply -f examples/cluster/enable-pod-identity.yaml
```

Wait the control plane is up:

```console
kubectl get cluster -w
```

Retrieve the cluster's Kubeconfig:

```console
clusterctl get kubeconfig pod-identity > /root/.kube/pod-identity.kubeconfig
```

Check node state:

```console
kubectl --kubeconfig=/root/.kube/pod-identity.kubeconfig get nodes
```

### Setup AWS pod identity webhook

[Amazon EKS Pod Identity Webhook](https://github.com/aws/amazon-eks-pod-identity-webhook) is for mutating pods that will require AWS IAM access.

AWS pod identity webhook requires cert-manager, (See [cert-manager installation](https://cert-manager.io/docs/installation/)).

```console
kubectl apply --kubeconfig=/root/.kube/pod-identity.kubeconfig -f https://github.com/cert-manager/cert-manager/releases/download/v1.11.0/cert-manager.yaml
```

Install AWS pod identity webhook:

```console
kubectl apply --kubeconfig=/root/.kube/pod-identity.kubeconfig -f examples/aws-pod-identity/pod-indentity.yaml
```

Now, the cluster is ready for use, try with [Load Balancer Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller/v2.4/deploy/installation/).

## Cleanup

{{< boilerplate cleanup >}}
