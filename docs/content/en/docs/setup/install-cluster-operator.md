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

Install cluster operator into the management cluster.

{{< boilerplate install-cluster-operator >}}

## Install cluster operator from release package


1. Go to [Kurator release](https://github.com/kurator-dev/kurator/releases) page to download the release package for your OS and extract.

    ```console
    curl -L https://github.com/kurator-dev/kurator/releases/download/{{< kurator-version >}}/kurator-{{< kurator-version >}}.tar.gz
    tar -zxvf kurator-{{< kurator-version >}}.tar.gz
    ```

1. Move to release package directory.

    ```console
    cd kurator-{{< kurator-version >}}
    ```

1. Install cluster operator into the management cluster.

    {{< boilerplate install-cluster-operator >}}

## Try to deploy a cluster with cluster operator

[Deploy cluster with kurator cluster operator](/docs/cluster-operator/kurator-cluster-api).

## Cleanup

```bash
helm uninstall kurator-cluster-operator -n kurator-system
```
