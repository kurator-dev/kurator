---
title: "Install Fleet Manager"
linkTitle: "Install Fleet Manager"
weight: 30
description: >
  Instructions on installing fleet manager.
---

## Prerequisites

Fleet manager depends on cluster operator, so refer to [Cluster operator installation guide](/docs/setup/install-cluster-operator).

## Install fleet manager from source

Build fleet manager image and helm chart:

```console
VERSION={{< kurator-version >}} make docker
VERSION={{< kurator-version >}} make gen-chart
```

Load image to kind cluster:

```console
kind load docker-image ghcr.io/kurator-dev/fleet-manager:{{< kurator-version >}} --name kurator-host
```

Install fleet manager into the management cluster.

{{< boilerplate install-fleet-manager >}}

## Install fleet manager from release package


1. Go to [Kurator release](https://github.com/kurator-dev/kurator/releases) page to download the release package for your OS and extract.

    ```console
    curl -L https://github.com/kurator-dev/kurator/releases/download/{{< kurator-version >}}/kurator-{{< kurator-version >}}.tar.gz
    tar -zxvf kurator-{{< kurator-version >}}.tar.gz
    ```

1. Move to release package directory.

    ```console
    cd kurator-{{< kurator-version >}}
    ```

1. Install fleet manager into the management cluster.

    {{< boilerplate install-fleet-manager >}}

## Try to create a fleet with fleet manager

[Get Started with Kurator Fleet](/docs/fleet-manager/create-fleet).

## Cleanup

```bash
helm uninstall kurator-cluster-operator -n kurator-system
helm uninstall kurator-fleet-manager -n kurator-system
```
