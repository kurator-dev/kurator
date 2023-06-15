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

1. Kurator metric depends on [Thanos](https://thanos.io), [Object Storage](https://thanos.io/tip/thanos/storage.md/) is required for Thanos. In the task, [Minio](https://min.io/) is used, setup by the [installation guide](/docs/setup/install-minio).

1. Running the following command to create two secrets to access attached clusters.

```console
kubectl create secret generic kurator-member1 --from-file=kurator-member1.config=/root/.kube/kurator-member1.config
kubectl create secret generic kurator-member2 --from-file=kurator-member2.config=/root/.kube/kurator-member2.config
```

### Create a fleet with metric plugin enabled

```console
kubectl apply -f examples/fleet/metric/metric-plugin.yaml
```

After a while, we can see the fleet is `ready`:

```console
kubectl wait fleet quickstart --for='jsonpath='{.status.phase}'=Ready'
```

Thanos and Grafana are installed correctly:

```console
kubectl get po 
NAME                                    READY   STATUS    RESTARTS   AGE
default-thanos-query-5b6d4dcf89-xm54l   1/1     Running   0          1m
default-thanos-storegateway-0           1/1     Running   0          1m
grafana-7b4bc74fcc-bvwgv                1/1     Running   0          1m
```

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
    gitRepository:
      interval: 3m0s
      ref:
        branch: main
      timeout: 1m0s
      url: https://github.com/kurator-dev/kurator
  syncPolicies:
    - destination:
        fleet: quickstart
      kustomization:
        interval: 5m0s
        path: ./examples/fleet/metric/monitor-demo
        prune: true
        timeout: 2m0s
EOF
```

## Query metric from Grafana

After a while, you can go to grafana datasource page query avalanche metric, it will looks like following:

{{< image width="100%"
    link="./image/grafana-thanos.jpeg"
    >}}

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
