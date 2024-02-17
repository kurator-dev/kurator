---
title: "Kurator E2E Test"
linkTitle: "Kurator E2E Test"
weight: 30
description: >
  Kurator End2End Test Guide
---

Kurator has provided E2E Test in order to avoid potential impacts of future changes on the overall project, reduce future maintenance costs, and improve code and architecture quality.

## Preparation For running E2E Test

### Install Kind

```console
# For AMD64 / x86_64
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64

# For ARM64
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-arm64

chmod +x ./kind

sudo mv ./kind /usr/local/bin/kind
```

### Install Helm

- Manual installation

```console
Can download the required version of helm you need from https://github.com/helm/helm/releases

To extract the zip file of helm, run: 
    tar -zxvf helm-vXXX-linux-amd64.tar.gz

Find the helm program in the extracted directory and move it to the desired directory, run:
    mv linux-amd64/helm /usr/local/bin/helm
```

- Script Installation

```console
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3

chmod 700 get_helm.sh

./get_helm.sh
```

## Steps For Running E2E Test

- Install Kubernetes Test Cluster Environment.
  
  ```console
  bash hack/e2e-test/build-cluster.sh
  ```

- Deploy kurator.
  
  ```console
  bash hack/e2e-test/install-kurator.sh
  ```

- Run Kurator E2E Tests.
  
  ```console
   bash hack/e2e-test/run-e2e.sh
  ```
