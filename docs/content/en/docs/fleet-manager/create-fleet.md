---
title: "Get Started with Kurator Fleet"
linkTitle: "Get Started with Kurator Fleet"
weight: 10
description: >
  The easiest way to manage multi clusters with fleet manager.
---

In this tutorial we’ll cover the basics of how to use [Fleet](https://kurator.dev/docs/references/fleet-api/#fleet) to manage a group of clusters.

## Prerequisites

Now fleet manager can only manage clusters that are created using kurator [Cluster API](https://kurator.dev/docs/references/cluster-api/#cluster). Please refer to [Get started with Kurator Cluster API](/docs/cluster-operator/kurator-cluster-api) to create a vanilla kubernetes cluster.

## Create a Fleet

We can create a fleet either with empty or pre-built clusters. Here in this example, we create a fleet with a cluster provisioned with kurator [cluster operator](/docs/cluster-operator).


### Apply the fleet manifest

```console
kubectl apply -f examples/fleet/fleet.yaml
```

After a while, we can see a pod `quickstart-init` running, which is to startup fleet control plane. It is not based on [karmada](https://karmada.io/)

When `quickstart-init` completed, we can see karmada components are running up.

```console
$ kubectl get pod
NAME                                            READY   STATUS      RESTARTS   AGE
etcd-0                                          1/1     Running     0          3m51s
karmada-aggregated-apiserver-79c684855c-nwh7z   1/1     Running     0          3m16s
karmada-apiserver-5648fbf56d-b7s25              1/1     Running     0          3m48s
karmada-controller-manager-7bcc575bdf-f6n8v     1/1     Running     0          2m57s
karmada-scheduler-7fbf87c489-tn7b9              1/1     Running     0          2m58s
karmada-webhook-8469dbf4c9-jcztq                1/1     Running     0          2m56s
kube-controller-manager-bbfdb8869-hmq2v         1/1     Running     0          2m59s
quickstart-init                                 0/1     Completed   0          3m54s
```

Now we can acquire the fleet entry from its status, the fleet has one cluster `quickstart` registered successfully. 
And the access credential is stored in `kubeconfig` secret.

```console
$ kubectl get fleet quickstart -n test -oyaml
apiVersion: fleet.kurator.dev/v1alpha1
kind: Fleet
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"fleet.kurator.dev/v1alpha1","kind":"Fleet","metadata":{"annotations":{},"name":"quickstart","namespace":"test"},"spec":{"clusters":[{"name":"quickstart"}]}}
  creationTimestamp: "2023-04-10T02:24:12Z"
  finalizers:
  - fleet.kurator.dev
  generation: 1
  name: quickstart
  namespace: test
  resourceVersion: "6317753"
  uid: 5483c4d2-0ccf-48f8-afae-945b18dac8d9
spec:
  clusters:
  - name: quickstart
status:
  credentialSecret: kubeconfig
  phase: Ready
  readyClusters: 1
```

### Check cluster registered

First acquire the kubeconfig from secret:

```console
kubectl get secret kubeconfig -o="jsonpath={.data.kubeconfig}" |base64 -d >kubeconfig
```

Now change the server address according to your real env, here we run the management cluster with `kind`. And the control plane api server service type is `NodePort`, so we replace control plane api endpoint with the node address:nodeport that can be accessed from outside of the cluster.

```console
sed -i "s|https:\/\/karmada-apiserver.test.svc.cluster.local:5443|https:\/\/172.18.0.3:32443|g"  kubeconfig
```

If everything workls well, you can check from the fleet control plane the `quickstart` cluster has been registered successfully.

```console
$ kubectl get clusters --kubeconfig=./kubeconfig
NAME         VERSION   MODE   READY   AGE
quickstart   v1.23.0   Push   True    1h
```

## Cleanup

Delete the fleet created

```console
kubectl delete fleet quickstart
```

Uninstall fleet manager:

```console
helm uninstall kurator-fleet-manager -n kurator-system
```

{{< boilerplate cleanup >}}
