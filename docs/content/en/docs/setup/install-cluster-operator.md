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

Build docker image and helm chart

    {{< boilerplate build-from-source >}}


Change directory to the helm charts

    ```console
    cd out/charts/
    ```

Install cluster operator into the management cluster.

    {{< boilerplate install-cluster-operator >}}


## Install cluster operator from release package


Go to [Kurator release](https://github.com/kurator-dev/kurator/releases) page to download the release package for your OS and extract.

    ```console
    curl -L https://github.com/kurator-dev/kurator/releases/download/v{{< kurator-version >}}/cluster-operator-{{< kurator-version >}}.tgz
    ```

Install cluster operator into the management cluster.

    {{< boilerplate install-cluster-operator >}}

## Install cluster operator from helm repo

Configure the Helm repository:

    ```console
    helm repo add kurator https://kurator-dev.github.io/helm-charts
    helm repo update
    ```

Install cluster operator into the management cluster.

    {{< boilerplate install-cluster-operator >}}

## Try to deploy a cluster with cluster operator

[Deploy cluster with kurator cluster operator](/docs/cluster-operator/kurator-cluster-api).

## Cleanup

```bash
helm uninstall kurator-cluster-operator -n kurator-system
```
