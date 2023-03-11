```console
# Please make sure cert manager is ready before install cluster operator
helm install --create-namespace kurator-cluster-operator out/charts/cluster-operator-{{< kurator-version >}}.tgz -n kurator-system
```
