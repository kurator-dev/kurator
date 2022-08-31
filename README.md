# Kurator

Kurator is an open source distributed cloud native platform that helps users to build their own distributed cloud native infrastructure and facilitates enterprise digital transformation.

Kurator integrates popular cloud native software stacks including [Karmada](https://github.com/karmada-io/karmada), [KubeEdge](https://github.com/kubeedge/kubeedge), [Volcano](https://github.com/volcano-sh/volcano), [Kubernetes](https://github.com/kubernetes/kubernetes), [Istio](https://github.com/istio/istio), [Prometheus](https://github.com/prometheus/prometheus), [ArgoCD](https://github.com/argoproj/argo-cd), [Pixie](https://github.com/pixie-io/pixie), etc.
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

### 1. Local env setup

```console
hack/local-dev-setup.sh
```

This script will create three clusters for you, one is used to host Karmada control plane and the other two will be joined as member clusters.


### 2. Install Karmada

**Install Karmada control plane:**

```console
kurator install karmada --kubeconfig=/root/.kube/kurator-host.config
```

**Join cluster `member1`:**

```console
kurator join karmada member1 \
    --cluster-kubeconfig=/root/.kube/kurator-members.config \
    --cluster-context=kurator-member1
```

**Join cluster `member2`:**

```console
kurator join karmada member2 \
    --cluster-kubeconfig=/root/.kube/kurator-members.config \
    --cluster-context=kurator-member2
```

### 3. Install Istio

```console
kurator install istio --primary member1 --remote member2
```

### 4. Install KubeEdge

**Install KubeEdge control plane:**

```console
kurator install kubeedge --cluster member1 --advertise-address=<ip>
```

**Join edge node:**

```console
kurator join edge  --cluster member1 \
    --cloudcore-address=<ip:port> \
    --node-ip= <node ip>\
    -p="${NODE_PWD}"
```

### 5. Install Volcano

```console
kurator install volcano
```

### 6. Install Prometheus

```console
kurator install prometheus --primary member1
```


## Contributing

If you're interested in being a contributor and want to get involved in
developing the Kurator code, please see [CONTRIBUTING](CONTRIBUTING.md) for
details on submitting patches and the contribution workflow.

## License

Kurator is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.
