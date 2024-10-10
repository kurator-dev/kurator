---
title: "Install Rollout Plugin"
linkTitle: "Install Rollout Plugin"
weight: 10
description: >
  Configure rollout plugin in fleet to enable kurator's rollout capability.
---

To support Kurator's Rollout, it's imperative to first configure the Rollout plugin for [Fleet](https://kurator.dev/docs/references/fleet-api/#fleet). This guide will walk you through configuring the [Flagger](https://docs.flagger.app/)-based rollout plugin for Fleet, laying the groundwork for Kurator's Rollout capabilities.

## Prerequisites

### 1. Fleet Manager Setup

Set up the Fleet manager by following the instructions in the [installation guide](/docs/setup/install-fleet-manager/).

### 2. Secrets and Setup for [Attached Clusters](https://kurator.dev/docs/fleet-manager/manage-attachedcluster)

```console
kubectl create secret generic kurator-member1 --from-file=kurator-member1.config=/root/.kube/kurator-member1.config
kubectl create secret generic kurator-member2 --from-file=kurator-member2.config=/root/.kube/kurator-member2.config
---
kubectl apply -f - <<EOF
apiVersion: cluster.kurator.dev/v1alpha1
kind: AttachedCluster
metadata:
  name: kurator-member1
  namespace: default
spec:
  kubeconfig:
    name: kurator-member1
    key: kurator-member1.config
---
apiVersion: cluster.kurator.dev/v1alpha1
kind: AttachedCluster
metadata:
  name: kurator-member2
  namespace: default
spec:
  kubeconfig:
    name: kurator-member2
    key: kurator-member2.config
EOF
```

## Create a Fleet with the Rollout Plugin Enabled

Run following command to create Flagger in the Fleet:

```console
kubectl apply -f -<<EOF
apiVersion: fleet.kurator.dev/v1alpha1
kind: Fleet
metadata:
  name: quickstart
  namespace: default
spec:
  clusters:
    - name: kurator-member1
      kind: AttachedCluster
    - name: kurator-member2
      kind: AttachedCluster
  plugin:
    flagger:
      publicTestloader: true
      trafficRoutingProvider: istio
EOF
```

### Fleet Rollout Plugin Configuration Explained

Let's delve into the `spec` section of the above Fleet:

- `clusters`: Contains the two `AttachedCluster` objects created earlier, indicating that the Rollout plugin will be installed on these two clusters.
- `plugin`: The `flagger` indicates the description of a Rollout plugin. It contains configurations for whether to install `publicTestloader` and `trafficRoutingProvider`.
  
    - `publicTestloader`: Indicates whether to install a common test loader to generate test traffic for application services.
    - `trafficRoutingProvider`: Traffic Routing Provider. Currently it supports Istio,Kuma and Nginx , in the future it will add support for other service meshes or ingress controllers. For example, Linkerd, Gloo, etc.

For more configuration information, please refer to the [Fleet API](https://kurator.dev/docs/references/fleet-api/).

## Verify the Installation

To ensure that the Rollout plugin is successfully installed and running.

Run the following commands:

```console
kubectl get pod -n istio-system --kubeconfig=/root/.kube/kurator-member1.config
kubectl get pod -n istio-system --kubeconfig=/root/.kube/kurator-member2.config
```

Initially, you should observe:

```console
istio-system-flagger-kurator-member1-649c65bd7d-ptgbj             1/1     Running   0          2m21s
istio-system-testloader-kurator-member1-loadtester-7ff7d75grwsh   1/1     Running   0          2m21s

```

## Cleanup

This section guides you through the process of cleaning up the fleets and plugins.

### 1. Cleanup the Rollout Plugin

If you only need to remove the Rollout plugin, simply edit the current fleet and remove the corresponding description:

```console
kubectl edit fleet.fleet.kurator.dev quickstart
```

To check the results of the deletion, you can observe that the Flagger components have been removed:

```console
kubectl get po -A --kubeconfig=/root/.kube/kurator-member1.config
kubectl get po -A --kubeconfig=/root/.kube/kurator-member2.config
```

If you wish to reinstall the components later, you can simply edit the fleet and add the necessary configurations.

### 2. Cleanup the Fleet

When the fleet is deleted, all associated plugins will also be removed:

```console
kubectl delete fleet.fleet.kurator.dev quickstart
```

### 3. Cleanup the fleet-manager (optional)

Run the following commands to delete fleet-manager and cluster-operator:

```console
helm uninstall kurator-cluster-operator -n kurator-system
helm uninstall kurator-fleet-manager -n kurator-system
```

Run the following commands to delete Fluxcd:

```console
helm delete fluxcd -n fluxcd-system
kubectl delete ns fluxcd-system --ignore-not-found
```
