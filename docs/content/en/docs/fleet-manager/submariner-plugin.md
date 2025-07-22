---
title: "Enable multi cluster networking with Submariner"
linkTitle: "Enable multi cluster networking with Submariner"
weight: 20
description: >
  The easiest way to manage multi cluster submariner plugin with fleet.
---

In this tutorial we’ll cover the basics of how to use [Fleet](https://kurator.dev/docs/references/fleet-api/#fleet) to manage submariner plugin on a group of clusters.

## Prerequisites

1. Setup Fleet manager by following the instructions in the [installation guide](/docs/setup/install-fleet-manager/).

2. Running the following command to create two secrets to access attached clusters.

```bash
# Create secrets for the attached clusters
kubectl create secret generic kurator-member1 --from-file=kurator-member1.config=/root/.kube/kurator-member1.config
kubectl create secret generic kurator-member2 --from-file=kurator-member2.config=/root/.kube/kurator-member2.config

# Label a gateway node for each attached cluster
kubectl label node kurator-member1-control-plane submariner.io/gateway=true --kubeconfig=/root/.kube/kurator-member1.config
kubectl label node kurator-member2-control-plane submariner.io/gateway=true --kubeconfig=/root/.kube/kurator-member2.config
```

### Create a fleet with metric plugin enabled

`SUBMARINER_PSK` needs to be set as [described](https://submariner.io/operations/deployment/helm/).

```bash
export SUBMARINER_PSK=$(LC_CTYPE=C tr -dc 'a-zA-Z0-9' < /dev/urandom | fold -w 64 | head -n 1)

envsubst < examples/fleet/network/submariner-plugin.yaml | kubectl apply -f -
```

After a while, we can see the fleet is `ready`:

```bash
kubectl wait fleet quickstart --for='jsonpath='{.status.phase}'=Ready'
```

### Verify the Installation

To ensure that the Submariner plugin is successfully installed and running.

Run the following commands:

```bash
kubectl get pod -n submariner-operator --kubeconfig=/root/.kube/kurator-member1.config
kubectl get pod -n submariner-operator --kubeconfig=/root/.kube/kurator-member2.config
```

More detailed verification steps can be done as follows:

> `subctl` needs to be installed to perform checks, please refer to the [Install subctl](https://submariner.io/operations/deployment/helm/#install-subctl).

- Diagnostic checks:

Perform `subctl diagnose` on each cluster:

```bash
subctl diagnose all --kubeconfig /root/.kube/kurator-member1.config
```

- Verify the connectivity between the clusters:

```bash
export KUBECONFIG=/root/.kube/kurator-member1.config:/root/.kube/kurator-member2.config
subctl verify --context kurator-member1 --tocontext kurator-member2
subctl verify --context kurator-member2 --tocontext kurator-member1
```

## Cleanup

Guides for you to clean up the fleets and plugins.

### 1. Cleanup the Submariner Plugin

Tutorial for manual uninstallation can be found in [Sumariner Documatation](https://submariner.io/operations/cleanup/#manual-uninstall).

> Deleting the cluster to start from scratch is **recommended** because potential crd dependencies conflicts and some hard-to-find legacy resources or settings.

### 2. Cleanup the Fleet

When the fleet is deleted, all associated plugins will also be removed:

```bash
kubectl delete fleet quickstart
```

### 3. Cleanup the Infrastructure

Uninstall fleet manager:

```bash
helm uninstall kurator-fleet-manager -n kurator-system
```

{{< boilerplate cleanup >}}
