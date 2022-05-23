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

## clean

```console
hack/local-dev-down.sh
```