```console
helm install --create-namespace kurator-fleet-manager out/charts/fleet-manager-{{< kurator-version >}}.tgz -n kurator-system
```

After a while, you can see kurator fleet manager running.

```bash
$ kubectl get pod -l app.kubernetes.io/name=kurator-fleet-manager -n kurator-system
NAME                                    READY   STATUS    RESTARTS   AGE
kurator-fleet-manager-d587f54b6-d4ldd   1/1     Running   0          53s
```
