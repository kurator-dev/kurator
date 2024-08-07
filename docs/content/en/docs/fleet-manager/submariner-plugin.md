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
kubectl create secret generic kurator-member1 --from-file=kurator-member1.config=/root/.kube/kurator-member1.config
kubectl create secret generic kurator-member2 --from-file=kurator-member2.config=/root/.kube/kurator-member2.config
```

### Create a fleet with metric plugin enabled

```bash
kubectl apply -f examples/fleet/network/submariner-plugin.yaml
```

After a while, we can see the fleet is `ready`:

```bash
kubectl wait fleet quickstart --for='jsonpath='{.status.phase}'=Ready'
```

### Verify the Installation

To ensure that the Submariner plugin is successfully installed and running.

Run the following commands:

```bash
kubectl get pod --kubeconfig=/root/.kube/kurator-member1.config | grep submariner
kubectl get pod --kubeconfig=/root/.kube/kurator-member2.config | grep submariner
```

## Cleanup

Delete the fleet created

```bash
kubectl delete fleet quickstart
```

Uninstall fleet manager:

```bash
helm uninstall kurator-fleet-manager -n kurator-system
```

{{< boilerplate cleanup >}}
