```console
# Please make sure cert manager is ready before install cluster operator
helm install --create-namespace kurator-cluster-operator out/charts/cluster-operator-0.3-dev.tgz -n kurator-system
```
