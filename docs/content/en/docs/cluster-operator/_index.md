---
title: "Managed your clusters with Kurator"
linkTitle: "Cluster operator"
weight: 3
description: >
  Learn how to get managed your clusters with Kurator.
---

Kurator cluster operator provide an easy way to get your Kubernetes cluster up and running. 

- Think of it as `operator` for clusters
- Built on [Cluster API](https://cluster-api.sigs.k8s.io) and [KubeSpray](https://kubespray.io/)
- Manage your cluster plugins, such as CNI, CSI and Ingress controller

## Concepts
  
- Kurator will provision all necessary resources (e.g. vpc, iam profile etc) for a Cluster;
- After Cluster API resources created, controllers imported from Custer API projects will provision the Kubernetes cluster;
- After cluster is provisioned, Kurator will install plugins for it;
- Once all plugins are properly installed, the cluster will be ready.
      
## Architecture

The overall architecture of Kurator cluster operator is shown as below:

{{< image width="100%"
    link="./image/cluster-operator.svg"
    >}}

The Kurator Cluster Operator runs various controllers, which watch cluster api objects and then talk to the underlying clusters' API servers to create Kubernetes clusters.

1. Cluster API controllers: imports from [Cluster API](https://cluster-api.sigs.k8s.io/introduction.html) projects and uses Kubernetes-style APIs and patterns to automate cluster lifecycle management for platform operators.
2. Kurator Cluster controller: watches Kurator Cluster API objects. When a Cluster object is added, the controller will provision target cluster.

