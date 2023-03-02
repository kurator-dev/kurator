```console
# Please make sure cert manager is ready before install cluster operator
kubectl create namespace kurator-system
helm install kurator-base out/charts/base-0.1.0.tgz -n kurator-system
helm install -n kurator-system kurator-cluster-operator out/charts/cluster-operator-0.1.0.tgz
```