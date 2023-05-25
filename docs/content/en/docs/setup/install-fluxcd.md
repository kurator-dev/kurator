---
title: "Install Fluxcd"
linkTitle: "Install Fluxcd"
weight: 40
description: >
  Instructions on installing Flucd.
---

## Install Fluxcd with Helm

Kurator use helm chart from fluxcd community, more details can be found [here](https://github.com/fluxcd-community/helm-charts).

Setup [Source Controller](https://fluxcd.io/flux/components/source/) and [Helm Controller](https://fluxcd.io/flux/components/helm/) with following command:

```console
helm repo add fluxcd-community https://fluxcd-community.github.io/helm-charts

cat <<EOF | helm install fluxcd fluxcd-community/flux2 --version 2.7.0 -n fluxcd-system --create-namespace -f -
imageAutomationController:
  create: false
imageReflectionController:
  create: false
kustomizeController:
  create: false
notificationController:
  create: false
EOF
```

Check the controller status:

```console
kubectl get po -n fluxcd-system
```

## Cleanup

```bash
helm delete fluxcd -n fluxcd-system
kubectl delete ns fluxcd-system --ignore-not-found
```
