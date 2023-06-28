Install cluster operator into the management cluster.

```console
helm install --create-namespace kurator-cluster-operator cluster-operator-{{< kurator-version >}}.tgz -n kurator-system
```

Verify the cluster operator chart installation:

```bash
$ kubectl get pod -l app.kubernetes.io/name=kurator-cluster-operator -n kurator-system
NAME                                        READY   STATUS    RESTARTS   AGE
kurator-cluster-operator-5977486c8f-7b5rc   1/1     Running   0          21h
```

