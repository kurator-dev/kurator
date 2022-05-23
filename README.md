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
out/linux-amd64/ubrain install karmada --kubeconfig=/root/.kube/ubrain-host.config
```

Join cluster member1:
```
out/linux-amd64/ubrain join karmada member1 --kubeconfig=/etc/karmada/karmada-apiserver.config \
    --cluster-kubeconfig=/root/.kube/ubrain-members.config \
    --cluster-context=ubrain-member1
```

Join cluster member2:
```
out/linux-amd64/ubrain join karmada member2 --kubeconfig=/etc/karmada/karmada-apiserver.config \
    --cluster-kubeconfig=/root/.kube/ubrain-members.config \
    --cluster-context=ubrain-member2
```

Verify:
```console
kubectl  --kubeconfig /etc/karmada/karmada-apiserver.config get clusters
```

## Install istio

```
out/linux-amd64/ubrain install istio --kubeconfig=/etc/karmada/karmada-apiserver.config --primary member1 --remote member2
```

### Install KubeEdge

```
out/linux-amd64/ubrain install kubeedge --kubeconfig=/etc/karmada/karmada-apiserver.config --cluster member1 --advertise-address="159.138.154.244"
```

```
out/linux-amd64/ubrain join edge --kubeconfig=/etc/karmada/karmada-apiserver.config --cluster member1 \
    --cloudcore-address="159.138.154.244:10000" \
    --node-ip="159.138.129.168" \
    -p="${NODE_PWD}"
```

## clean

```console
hack/local-dev-down.sh
```