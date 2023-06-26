---
title: "Install Fleet Manager"
linkTitle: "Install Fleet Manager"
weight: 30
description: >
  Instructions on installing fleet manager.
---

## Prerequisites

Fleet manager depends on cluster operator, so refer to [Cluster operator installation guide](/docs/setup/install-cluster-operator).

## Install FluxCD with Helm

Fleet manager depends on [Fluxcd](https://fluxcd.io/flux/), Kurator use helm chart from fluxcd community, more details can be found [here](https://github.com/fluxcd-community/helm-charts).

Setup with following command:

```console
helm repo add fluxcd-community https://fluxcd-community.github.io/helm-charts

cat <<EOF | helm install fluxcd fluxcd-community/flux2 --version 2.7.0 -n fluxcd-system --create-namespace -f -
imageAutomationController:
  create: false
imageReflectionController:
  create: false
notificationController:
  create: false
EOF
```

Check the controller status:

```console
kubectl get po -n fluxcd-system
```

## Install fleet manager from source

Build docker image and helm chart

    {{< boilerplate build-from-source >}}


Change directory to the helm charts

    ```console
    cd out/charts/
    ```

Install fleet manager into the management cluster.

    {{< boilerplate install-fleet-manager >}}

## Install fleet manager from release package

Go to [Kurator release](https://github.com/kurator-dev/kurator/releases) page to download the release package for your OS and extract.

    ```console
    curl -L https://github.com/kurator-dev/kurator/releases/download/v{{< kurator-version >}}/fleet-manager-{{< kurator-version >}}.tgz
    ```

Install fleet manager into the management cluster.

    {{< boilerplate install-fleet-manager >}}


## Install fleet manager from helm repo

Configure the Helm repository:

    ```console
    helm repo add kurator https://kurator-dev.github.io/helm-charts
    helm repo update
    ```

Install fleet manager into the management cluster.

    {{< boilerplate install-fleet-manager >}}

## Try to create a fleet with fleet manager

[Get Started with Kurator Fleet](/docs/fleet-manager/create-fleet).

## Cleanup

```bash
helm uninstall kurator-cluster-operator -n kurator-system
helm uninstall kurator-fleet-manager -n kurator-system
```

```bash
helm delete fluxcd -n fluxcd-system
kubectl delete ns fluxcd-system --ignore-not-found
```
