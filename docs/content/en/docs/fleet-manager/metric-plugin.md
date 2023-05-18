---
title: "Enable multi cluster Monitoring with fleet"
linkTitle: "Enable multi cluster Monitoring with fleet"
weight: 20
description: >
  The easiest way to manage multi cluster metric plugin with fleet.
---

In this tutorial we’ll cover the basics of how to use [Fleet](https://kurator.dev/docs/references/fleet-api/#fleet) to manage metirc plugin on a group of clusters.

## Architecture

Fleet's multi cluster monitoring is built on top [Prometheus](https://prometheus.io/) and [Thanos](https://thanos.io/), the overall architecture is shown as below: 

{{< image width="100%"
    link="./image/fleet-metric.png"
    >}}

## Prerequisites

1. Setup Fleet manager by following the instructions in the [installation guide](/docs/setup/install-fleet-manager/).

1. Fleet manager depends on [Fluxcd](https://fluxcd.io/flux/), setup by following the instructions in the [installation guide](/docs/setup/install-fluxcd/).

1. Kurator metric depends on [Thanos](https://thanos.io), [Object Storage](https://thanos.io/tip/thanos/storage.md/) is required for Thanos. In the task, [Minio](https://min.io/) is used, setup by the [installation guide](/docs/setup/install-minio).


### Create a fleet with metric plugin enabled

```console
kubectl apply -f examples/fleet/metric-plugin.yaml
```

After a while, we can see the fleet is `ready`:

```console
kubectl wait fleet quickstart --for='jsonpath='{.status.phase}'=Ready'
```

Then, we can access Thanos web UI with `localhost:9090/stores` to verify status of stores:

```console
kubectl port-forward svc/default-thanos-query 9090:9090 --address 0.0.0.0
```

{{< image width="100%"
    link="./image/thanos-ui.jpeg"
    >}}

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
