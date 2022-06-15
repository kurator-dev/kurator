## Kurator

Kurator is an open source distributed cloud native platform that helps users to build their own distributed cloud native infrustucter and facilitates enterprise digital transformation.

Kurator integrates popular cloud native software stacks including [Karmada](https://github.com/karmada-io/karmada), [KubeEdge](https://github.com/kubeedge/kubeedge), [Volcano](https://github.com/volcano-sh/volcano), [Kubernetes](https://github.com/kubernetes/kubernetes), [Istio](https://github.com/istio/istio), [Prometheus](), etc. 
It provides powerful capabilities to multi-cloud and multi-cluster, including:

- Multi-cloud, Edge-cloud, Edge-edge Synergy
- Unified Resource Orchestration
- Unified Scheduling
- Unified Traffic Management
- Unified Telemetry

## Quick start

This guide will cover:
- Install Karmada and join a Kubernetes member cluster
- Install Istio
- Install KubeEdge and join an edge node
- Install Volcano
- Install Prometheus

### Local env setup

```console
$ hack/local-dev-setup.sh
```

This script will create three clusters for you, one is used to host Karmada control plane and the other two will be joined as member clusters.


### Install Karmada

**Install Karmada control plane:**

```console
$ kurator install Karmada --kubeconfig=/root/.kube/kurator-host.config
```

**Join cluster `member1`:**

```console
$ kurator join Karmada member1 \
    --cluster-kubeconfig=/root/.kube/kurator-members.config \
    --cluster-context=kurator-member1
```

**Join cluster `member2`:**

```console
$ kurator join Karmada member2 \
    --cluster-kubeconfig=/root/.kube/kurator-members.config \
    --cluster-context=kurator-member2
```

### Install Istio

```console
$ kurator install istio --primary member1 --remote member2
```

### Install KubeEdge

**Install KubeEdge control plane**

```console
$ kurator install kubeedge --cluster member1 --advertise-address=<ip>
```

**Join edge node**
```console
$ kurator join edge  --cluster member1 \
    --cloudcore-address=<ip:port> \
    --node-ip= <node ip>\
    -p="${NODE_PWD}"
```

### Install Volcano

```console
$ kurator install volcano
```

### Install Prometheus

```console
$ kurator install prometheus --primary member1
```
