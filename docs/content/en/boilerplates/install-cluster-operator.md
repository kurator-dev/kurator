Install cluster operator into the management cluster.

    ```console
    helm install --create-namespace kurator-cluster-operator cluster-operator-{{< kurator-version >}}.tgz -n kurator-system
    ```

Verify the cluster operator chart installation:

    ```bash
    $ kubectl get pod -n kurator-system
    NAME                                        READY   STATUS    RESTARTS   AGE
    kurator-cluster-operator-84d64c89db-brmv2   1/1     Running   0          14s
    ```

