# Cluster Operator

## Quickstart

- Build

```
make docker
make gen-chart

kind load docker-image ghcr.io/kurator-dev/cluster-operator:latest --name <your_kind_cluster>
```

- Install cert manager

```
helm repo add jetstack https://charts.jetstack.io
helm repo update
kubectl create namespace cert-manager
helm install -n cert-manager cert-manager jetstack/cert-manager --set installCRDs=true
```

- Install cluster operator

```
# must run this after cert-manager is ready
kubectl create namespace kurator-system
helm install kurator-base out/charts/base-0.1.0.tgz -n kurator-system
helm install kurator-cluster-operator out/charts/cluster-operator-0.1.0.tgz -n kurator-system
```

## Cleanup

```
helm uninstall kurator-base -n kurator-system
helm uninstall kurator-cluster-operator -n kurator-system 
kubectl delete crd $(k get crds | grep cluster.x-k8s.io | awk '{print $1}')
kubectl delete ns kurator-system
```