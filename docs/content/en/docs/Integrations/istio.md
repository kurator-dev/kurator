---
title: "Integrate Istio with Kurator"
description: >
    This task shows how to integrate service mesh with Kurator.
---

### What is Istio

[Istio](https://istio.io) is a service mesh—a modernized service networking layer that provides a transparent and language-independent way to flexibly and easily automate application network functions.

In this task, we will show you how to install Istio with kurator.

### Prerequisites

{{% readfile "prerequisites-karmada.md" %}}

### Install Istio

Kurator provides a very simple command to install Istio multicluster [Primary-Remote](https://istio.io/latest/docs/setup/install/multicluster/primary-remote/) model and add karmada-apiserver as the destination to apply [Istio configurations](https://istio.io/latest/docs/reference/config/) to.

- `--primary` specifies the cluster where the istio control plane install.
- `--remote` specifies the cluster names that are managed by istio.

```bash
kurator install istio --primary member1 --remote member2
```

### Install Istio on different networks

Kurator also providers a simple way to install [Istio on different networks](https://istio.io/latest/docs/setup/install/multicluster/primary-remote_multi-network/).

First, you need label cluster to describe network topology. The following command will label clusters with different networks:

```shell
kubectl label cluster member1 topology.istio.io/network=network1 --overwrite --kubeconfig=/etc/karmada/karmada-apiserver.config
kubectl label cluster member2 topology.istio.io/network=network2 --overwrite --kubeconfig=/etc/karmada/karmada-apiserver.config
```

Now, run the following command to install Istio:

```bash
kurator install istio --primary member1 --remote member2
```

### Next steps

<!-- provider a simple way with kurator -->
Now, you can [verify the installion](https://istio.io/latest/docs/setup/install/multicluster/verify)
