---
title: "Unified Application Distribution with Kurator"
linkTitle: "Unified Application Distribution with Kurator"
weight: 17
description: >
  This is your go-to guide for distributing applications uniformly with Kurator.
---

In this guide, we will introduce how to use Kurator to distribute applications uniformly with [Fleet](https://kurator.dev/docs/references/fleet-api/#fleet).

## Architecture

Kurator offers a unified system for distributing applications across multiple clusters, powered by Fleet.

By making use of the GitOps approach through [FluxCD](https://fluxcd.io/flux/), Kurator automates the process of syncing and deploying applications. This makes the entire procedure quicker and more precise

Built to be flexible and responsive, Kurator's distribution system is specially designed to accommodate various business and cluster demands.

The overall architecture is shown as below:

{{< image width="100%"
    link="./image/fleet-application.svg"
    >}}

## Prerequisites

Setup Fleet manager by following the instructions in the [installation guide](/docs/setup/install-fleet-manager).

## Create an example application

The following example uses two locally attachedClusters for testing. For more details about attachedCluster, refer to the [manage attachedCluster](/docs/fleet-manage/manage-attachedcluster) page.

After the Kind cluster is ready, you can check the original status of the cluster for comparison with the following example.

```console
# Replace `/root/.kube/kurator-member1.config` with the actual path of your cluster's kubeconfig file.
kubectl get po -A --kubeconfig=/root/.kube/kurator-member1.config
```

Apply example application with the following command:

```console
$ kubectl apply -f examples/application/quickstart1.yaml
attachedcluster.cluster.kurator.dev/kurator-member1 created
attachedcluster.cluster.kurator.dev/kurator-member2 created
fleet.fleet.kurator.dev/quickstart created
application.apps.kurator.dev/quickstart1 created
```

Here is the content of example application resource.
The YAML configuration of the example application outlines its source, synchronization policy, and other key settings.
This includes the `gitRepository` as its source and two `kustomization` syncPolicies referring to a fleet that contains two attachedClusters

```console
apiVersion: apps.kurator.dev/v1alpha1
kind: Application
metadata:
  name: quickstart1
  namespace: default
spec:
  source:
    gitRepository:
      interval: 3m0s
      ref:
        branch: master
      timeout: 1m0s
      url: https://github.com/stefanprodan/podinfo
  syncPolicies:
    - destination:
        fleet: fleet-gitrepo
      kustomization:
        interval: 5m0s
        path: ./deploy/webapp
        prune: true
        timeout: 2m0s
    - destination:
        fleet: fleet-gitrepo
      kustomization:
        targetNamespace: default
        interval: 5m0s
        path: ./kustomize
        prune: true
        timeout: 2m0s
```

Optionally, you can also try testing the examples below with different combinations

```console
# This includes the `helmRepository` as source and the `helmRelease` as syncPolicies 
kubectl apply -f examples/application/quickstart2.yaml
```

```console
# This includes the `gitRepository` as source and the `helmRelease` as syncPolicies 
kubectl apply -f examples/application/quickstart3.yaml
```

## Verifying the unified application distribution result

After the fleet's control plane complete (approximately 1 minute), 
you can see within a few seconds that the application has been installed according to the configuration.

Use the following command to view the application installation result:

```console
$ kubectl get po -A --kubeconfig=/root/.kube/kurator-member1.config
NAMESPACE            NAME                                        READY   STATUS    RESTARTS       AGE
default              podinfo-588b784c7c-65sx7                    1/1     Running   0              4m41s
default              podinfo-588b784c7c-hh8j8                    1/1     Running   0              4m56s
...
webapp               backend-d4b8d7844-8l2vp                     1/1     Running   0              4m56s
webapp               frontend-6d94ff7cb5-m55pt                   1/1     Running   0              4m56s

$ kubectl get po -A --kubeconfig=/root/.kube/kurator-member2.config
NAMESPACE            NAME                                        READY   STATUS    RESTARTS       AGE
default              podinfo-588b784c7c-8bmtp                    1/1     Running   0              22m
default              podinfo-588b784c7c-cczfv                    1/1     Running   0              23m
...
webapp               backend-d4b8d7844-c6x8x                     1/1     Running   0              23m
webapp               frontend-6d94ff7cb5-4d4hh                   1/1     Running   0              23m
```

The command output lists all the pods deployed across two clusters. 
If the application distribution is successful, you should see pods from the 'podinfo' and 'webapp' applications installed in both clusters

## CleanUp

Use the following command to clean up the `quickstart1` application and related resources, like `gitRepository`, `kustomization`, `helmRelease`, etc.

```console
kubectl delete applications.apps.kurator.dev quickstart1
```

Also, you can confirm that the corresponding cluster applications have been cleared through the following command:

```console
# Replace `/root/.kube/kurator-member1.config` with the actual path of your cluster's kubeconfig file.
kubectl get po -A --kubeconfig=/root/.kube/kurator-member1.config
```
