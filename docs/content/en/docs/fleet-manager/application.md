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

Apply pre-prepared attachedClusters and fleet with the following command:

```console
$ kubectl apply -f examples/application/common/
attachedcluster.cluster.kurator.dev/kurator-member1 created
attachedcluster.cluster.kurator.dev/kurator-member2 created
fleet.fleet.kurator.dev/quickstart created
```

Apply example application with the following command:

```console
$ kubectl apply -f examples/application/gitrepo-kustomization-demo.yaml
application.apps.kurator.dev/gitrepo-kustomization-demo created
```

Here is the content of example application resource.
The YAML configuration of the example application outlines its source, synchronization policy, and other key settings.
This includes the `gitRepository` as its source and two `kustomization` syncPolicies referring to a fleet that contains two attachedClusters

```console
apiVersion: apps.kurator.dev/v1alpha1
kind: Application
metadata:
  name: gitrepo-kustomization-demo
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
        fleet: quickstart
      kustomization:
        interval: 5m0s
        path: ./deploy/webapp
        prune: true
        timeout: 2m0s
    - destination:
        fleet: quickstart
      kustomization:
        targetNamespace: default
        interval: 5m0s
        path: ./kustomize
        prune: true
        timeout: 2m0s
```

## Try other example application

Optionally, you can also try testing the examples below with different combinations

```console
# This includes the `helmRepository` as source and the `helmRelease` as syncPolicies 
kubectl apply -f examples/application/helmrepo-helmrelease-demo.yaml
```

Here is the configuration of application.

```console
apiVersion: apps.kurator.dev/v1alpha1
kind: Application
metadata:
  name: helmrepo-helmrelease-demo
  namespace: default
spec:
  source:
    helmRepository:
      interval: 5m
      url: https://stefanprodan.github.io/podinfo
  syncPolicies:
    - destination:
        fleet: quickstart
      helm:
        releaseName: podinfo
        chart:
          spec:
            chart: podinfo
        interval: 50m
        install:
          remediation:
            retries: 3
        values:
          redis:
            enabled: true
            repository: public.ecr.aws/docker/library/redis
            tag: 7.0.6
          ingress:
            enabled: true
            className: nginx
```

```console
# This includes the `gitRepository` as source and the `helmRelease` as syncPolicies 
kubectl apply -f examples/application/gitrepo-helmrelease-demo.yaml
```

Here is the configuration of application.

```console
apiVersion: apps.kurator.dev/v1alpha1
kind: Application
metadata:
  name: gitrepo-helmrelease-demo
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
        fleet: quickstart
      helm:
        releaseName: podinfo
        chart:
          spec:
            chart: ./charts/podinfo
        interval: 50m
        install:
          remediation:
            retries: 3
        values:
          redis:
            enabled: true
            repository: public.ecr.aws/docker/library/redis
            tag: 7.0.6
          ingress:
            enabled: true
            className: nginx
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

## Cluster Selection with Application Policies

You can add selectors to an application policy to ensure that the policy is applied specifically to corresponding clusters. This functionality is particularly useful in scenarios where a fleet contains various types of clusters. For instance, if your fleet includes clusters for testing and others for development, different application distribution strategies may be required.

Labels such as `env=dev` can be assigned to clusters, and the same selectors can then be specified in the corresponding application policy. Once configured, the application will select the specific clusters in the fleet based on these selectors to distribute the application.

Please note the following considerations:

1. You have the option to set a default selector for all policies under `application.spec.destination`, or to configure it within individual policies. Kurator gives precedence to the policy-level setting - it resorts to the default setting only when the destination within the policy is not set.

1. To ensure that the policy functions as expected, selectors should be added to the cluster prior to running the application.

Let's look at a use case:

We'll continue with the test fleet and attachedCluster used previously:

```console
kubectl apply -f examples/application/common/
```

Next, let's add labels to the attachedCluster:

```console
kubectl label attachedcluster kurator-member1 env=test
kubectl label attachedcluster kurator-member2 env=dev
```

To test the selector, run the application:

```console
kubectl apply -f examples/application/cluster-selector-demo.yaml
```

You can inspect the clusters with the following commands:

```console
kubectl get po -A --kubeconfig=/root/.kube/kurator-member1.config
kubectl get po -A --kubeconfig=/root/.kube/kurator-member2.config
```

Upon examining the respective clusters, you'll find that applications originating from the same source configuration have been distributed to different clusters based on their respective policy selector labels.

## Playgroud

Kurator uses killercoda to provide [applications demo](https://killercoda.com/965010e0-4f60-4a28-bf27-597d3kurator/scenario/application-example), allowing users to experience hands-on operations.

## CleanUp

Use the following command to clean up the `gitrepo-kustomization-demo` application and related resources, like `gitRepository`, `kustomization`, `helmRelease`, etc.

```console
kubectl delete applications.apps.kurator.dev gitrepo-kustomization-demo
```

Also, you can confirm that the corresponding cluster applications have been cleared through the following command:

```console
# Replace `/root/.kube/kurator-member1.config` with the actual path of your cluster's kubeconfig file.
kubectl get po -A --kubeconfig=/root/.kube/kurator-member1.config
```
