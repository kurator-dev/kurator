```console
# Please make sure cert manager is ready before install cluster operator
helm install --create-namespace kurator-cluster-operator out/charts/cluster-operator-{{< kurator-version >}}.tgz -n kurator-system
```

After a while, you can see kurator cluster operator running.

```bash
$ kubectl get pod -n kurator-system
NAME                                        READY   STATUS    RESTARTS   AGE
kurator-cluster-operator-84d64c89db-brmv2   1/1     Running   0          14s
```

