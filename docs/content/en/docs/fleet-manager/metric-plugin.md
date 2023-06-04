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
kubectl apply -f examples/fleet/metric/metric-plugin.yaml
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

## Apply more monitor settings with Fleet Application

Run following command to create a [avalanche](https://github.com/prometheus-community/avalanche) pod and ServiceMonitor in the fleet:

```console
cat <<EOF | kubectl apply -f -
apiVersion: apps.kurator.dev/v1alpha1
kind: Application
metadata:
  name: metric-demo
  namespace: default
spec:
  source:
    gitRepo:
      interval: 3m0s
      ref:
        branch: master
      timeout: 1m0s
      url: https://github.com/kurator-dev/kurator
  syncPolicy:
    - destination:
        fleet: quickstart
      kustomization:
        interval: 5m0s
        path: ./examples/fleet/metric/monitor-demo
        prune: true
        timeout: 2m0s
EOF
```

## Cleanup

Delete the fleet created

```console
kubectl delete application metric-demo
kubectl delete fleet quickstart
```

Uninstall fleet manager:

```console
helm uninstall kurator-fleet-manager -n kurator-system
```

{{< boilerplate cleanup >}}
