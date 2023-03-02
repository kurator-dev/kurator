***IMPORTANT: In order to ensure a proper cleanup of your infrastructure you must always delete the cluster object. Deleting the entire cluster template with kubectl delete -f capi-quickstart.yaml might lead to pending resources to be cleaned up manually.***

```console
kubectl delete cluster capi-quickstart
```

Uninstall cluster operator:

```console
helm uninstall kurator-cluster-operator -n kurator-system 
helm uninstall kurator-base -n kurator-system
kubectl delete ns kurator-system
```

*Optional*, unintall cert manager:

```console
helm uninstall -n cert-manager cert-manager
```


*Optional*, shutdown cluster:
```
kind delete cluster --name kurator
```
