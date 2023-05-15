---
title: "Install Karmada with Kurator"
description: >
    This task shows how to integrate multi-cluster orchestration with Kurator.
---

## Install Karmada

The documentation uses `Ubuntu 20.04.4 LTS` as an example.

### Prerequisites

Deploy a kubernetes cluster using kurator's scripts. This script will create three clusters for you, one is used to host Karmada control plane and the other two will be joined as member clusters.

```bash
git clone https://github.com/kurator-dev/kurator.git
cd kurator
hack/local-dev-setup.sh
```

### Deploy Karmada

Compile `kurator` from source

```bash
git clone https://github.com/kurator-dev/kurator.git
cd kurator
make kurator
```

Install karmada control plane

```bash
kurator install karmada --kubeconfig=/root/.kube/config
```

> When deploying kubernetes using a script, the kubeconfig is kurator-host.config

karmada installation parameters can be set with `--set`, e.g.

```bash
kurator install karmada --set karmada-data=/etc/Karmada-test --set port=32222 --kubeconfig .kube/config
```

### Add kubernetes cluster to karmada control plane

```bash
kurator join karmada member1 \
    --cluster-kubeconfig=/root/.kube/kurator-members.config \
    --cluster-context=kurator-member1
```

Show members of karmada 

```bash
kubectl --kubeconfig /etc/karmada/karmada-apiserver.config get clusters
```

>Notice
>
> karmada v1.2.0 and below version, does not support kubernetes v1.24.0 and above version join the karmada control plane
>
> For details, please see [1961](https://github.com/karmada-io/karmada/issues/1961)
