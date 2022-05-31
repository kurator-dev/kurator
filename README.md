# UCS-D

# Quick start

## Local setup

```console
hack/local-dev-setup.sh
```

## Build

```console
make
```

## Install karmada

Install karmada control plane:

```console
out/linux-amd64/kurator install karmada --kubeconfig=/root/.kube/kurator-host.config
```

Join cluster member1:
```
out/linux-amd64/kurator join karmada member1 --kubeconfig=/etc/karmada/karmada-apiserver.config \
    --cluster-kubeconfig=/root/.kube/kurator-members.config \
    --cluster-context=kurator-member1
```

Join cluster member2:
```
out/linux-amd64/kurator join karmada member2 --kubeconfig=/etc/karmada/karmada-apiserver.config \
    --cluster-kubeconfig=/root/.kube/kurator-members.config \
    --cluster-context=kurator-member2
```

Verify:
```console
kubectl  --kubeconfig /etc/karmada/karmada-apiserver.config get clusters
```

## Install istio

```
out/linux-amd64/kurator install istio --kubeconfig=/etc/karmada/karmada-apiserver.config --primary member1 --remote member2
```

### Install KubeEdge

```
out/linux-amd64/kurator install kubeedge --kubeconfig=/etc/karmada/karmada-apiserver.config --cluster member1 --advertise-address="159.138.154.244"
```

```
out/linux-amd64/kurator join edge --kubeconfig=/etc/karmada/karmada-apiserver.config --cluster member1 \
    --cloudcore-address="159.138.154.244:10000" \
    --node-ip="159.138.129.168" \
    -p="${NODE_PWD}"
```

## Install Prometheus

```
out/linux-amd64/kurator install prometheus --kubeconfig=/etc/karmada/karmada-apiserver.config

# install prometheus with federation
LOGGING_LEVEL=debug out/linux-amd64/kurator install prometheus --kubeconfig=/etc/karmada/karmada-apiserver.config --primary member1
```

## clean

```console
hack/local-dev-down.sh
```