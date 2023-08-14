---
title: "Install Cluster Operator"
linkTitle: "Install Cluster Operator"
weight: 20
description: >
  Instructions on installing cluster operator.
---

## Prerequisites

{{< boilerplate prerequisites >}}

## Install from source

{{< boilerplate build-from-source >}}


Change directory to the helm charts

```bash
cd out/charts/
```

{{< boilerplate install-cluster-operator >}}


## Install cluster operator from release package

Go to [Kurator release](https://github.com/kurator-dev/kurator/releases) page to download the release package for your OS and extract.

```bash
curl -LO https://github.com/kurator-dev/kurator/releases/download/v{{< kurator-version >}}/cluster-operator-{{< kurator-version >}}.tgz
```

{{< boilerplate install-cluster-operator >}}

## Install cluster operator from helm repo

Configure the Helm repository:

```bash
helm repo add kurator https://kurator-dev.github.io/helm-charts
helm repo update
```

Install cluster operator into the management cluster.

```console
helm install --create-namespace  kurator-cluster-operator kurator/cluster-operator --version={{< kurator-version >}} -n kurator-system 

```

Verify the cluster operator chart installation:

```bash
$ kubectl get pod -l app.kubernetes.io/name=kurator-cluster-operator -n kurator-system
NAME                                        READY   STATUS    RESTARTS   AGE
kurator-cluster-operator-5977486c8f-7b5rc   1/1     Running   0          21h
```


## Try to deploy a cluster with cluster operator

[Deploy cluster with kurator cluster operator](/docs/cluster-operator/kurator-cluster-api).

## Cleanup

```bash
helm uninstall kurator-cluster-operator -n kurator-system
```
