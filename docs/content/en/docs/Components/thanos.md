---
title: "Integrate Thanos with Kurator"
linkTitle: "Integrate Thanos with Kurator"
description: >
    Integrate Thanos with Kurator?
---

### What is Thanos

[Thanos](https://thanos.io) provides a global query view, high availability, data backup with historical, cheap data access as its core features in a single binary.

In this task, we will first show you how to install Thanos with kurator.

### Prerequisites

This task requires you have installed karmada and have joined at least one member cluster.
Otherwise, setup karmada environment following [Install Karmada with Kurator](./karmada.md).

### Install Thanos

Kurator provides a very simple command to install Thanos and add karmada-apiserver as the destination to deploy application to.
    
- `--kubeconfig` and `--context` specifies the cluster where Karmada Apiserver, Kurator use Karmada to orchestrate Thanos application.
- `--host-kubeconfig` and `--host-context` specifies the cluster where to deploy Thanos itself, it is the kurator host cluster that hold Thanos.
- `--object-store-config` specifies the object store configuration used by Thanos, more details can be found [here](https://prometheus-operator.dev/docs/operator/thanos/#configuring-thanos-object-storage).

```bash
kurator install thanos --host-kubeconfig /root/.kube/kurator-host.config --host-context kurator-host --object-store-config /root/thanos/thanos-config.yaml
```

### Verify Thanos query

Suppose you are running the installation in an external VM, you need to expose Thanos server. 

```bash
kubectl port-forward --address 0.0.0.0 svc/thanos-query -n thanos 9090:9090 --kubeconfig /root/.kube/kurator-host.config --context kurator-host
```

And then access Thanos server `https://<your vm address>:9090/stores` from your browser.

{{< image width="75%"
    link="./image/thanos.png"
    caption="thanos stores"
    >}}