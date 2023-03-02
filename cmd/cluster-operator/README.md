# Cluster Operator

## Quickstart

- Build

```
VERSION=0.3-dev make docker gen-chart

kind load docker-image ghcr.io/kurator-dev/cluster-operator:0.3-dev --name <your_kind_cluster>
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
helm install --create-namespace kurator-cluster-operator out/charts/cluster-operator-0.3-dev.tgz -n kurator-system

kubectl logs -l app.kubernetes.io/name=kurator-cluster-operator -n kurator-system --tail=-1
```

## Cleanup

```
helm uninstall kurator-cluster-operator -n kurator-system
kubectl delete crd $(kubectl get crds | grep cluster.x-k8s.io | awk '{print $1}')
kubectl delete crd $(kubectl get crds | grep kurator.dev | awk '{print $1}')
kubectl delete ns kurator-system
```