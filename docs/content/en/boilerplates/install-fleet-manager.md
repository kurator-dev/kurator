Verify the fleet manager chart installation:


```bash
$ kubectl get pod -l app.kubernetes.io/name=kurator-fleet-manager -n kurator-system
NAME                                    READY   STATUS    RESTARTS   AGE
kurator-fleet-manager-d587f54b6-d4ldd   1/1     Running   0          53s
```
