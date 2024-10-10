---
title: "Kuma Canary Deployment"
linkTitle: "Kuma Canary Deployment"
weight: 20
description: >
  A comprehensive guide on Kurator's Canary Deployment uses Kuma as ingress, providing an overview and quick start guide.
---

## prerequisites

In the subsequent sections, we'll guide you through a hands-on demonstration.

These are some of the prerequisites needed to use Kurator Rollout:

### Kubernetes Clusters

Kubernetes v1.27.3 or higher is supported.
  
You can use [Kind](https://kind.sigs.k8s.io/) to create clusters as needed.
It is recommended to use [Kurator's scripts](https://kurator.dev/docs/setup/install-cluster-operator/#setup-kubernetes-clusters-with-kind) to create multi-clusters environment.

Notes: You can find the mapping between Kind node image versions and Kubernetes versions on [Kind Release](https://github.com/kubernetes-sigs/kind/releases). Additionally, the website provides a lookup table showing compatible Kind and node image versions.

### Kuma

When Kuma is specified in fleet's `rollout.trafficRoutingProvider` , Kurator will install Kuma via helm in the fleet-managed clusters.

You can review the results a few minutes after applying fleet:

```console
kubectl get po -n kuma-system --kubeconfig=/root/.kube/kurator-member1.config
NAME                                                              READY   STATUS    RESTARTS   AGE
kuma-control-plane-748fbfd949-27nnk                               1/1     Running   0          80s
kuma-system-flagger-kurator-member-5cd65cdc48-srqmt               1/1     Running   0          5m5s
kuma-system-testloader-kurator-member-loadtester-7ff7d757bh5m4k   1/1     Running   0          5m5s


### Prometheus

You can install Prometheus paired with Kuma using the following commands:

```console
export KUBECONFIG=/root/.kube/kurator-member1.config
kumactl install observability --components "prometheus" | kubectl apply -f -
```

**Note:**Refer to [kuma documentation](https://docs.konghq.com/mesh/latest/production/install-kumactl/) for instructions on installing kumactl.

Review the results:

```console
kubectl get po -n mesh-observability --kubeconfig=/root/.kube/kurator-member1.config
NAME                                             READY   STATUS    RESTARTS   AGE
prometheus-kube-state-metrics-56b6556878-p4dj4   1/1     Running   0          5m2s
prometheus-server-7f4ddbb69-swz4w                3/3     Running   0          5m2s
```

### Mesh

To allow external programs access to the services you've deployed, you will need to create an Mesh resource.

You can create an mesh using the following commands:

```console
kubectl apply -f -<<EOF
apiVersion: kuma.io/v1alpha1
kind: Mesh
metadata:
  name: default
  namespace: kuma-system
spec:
  metrics:
    backends:
    - name: prometheus-1
      type: prometheus
    enabledBackend: prometheus-1
EOF
```

Review the results:

```console
kubectl get mesh --kubeconfig=/root/.kube/kurator-member1.config
           
NAME      AGE
default   12m

```

### Kurator Rollout Plugin

Before delving into the how to Perform a Unified Rollout, ensure you have successfully installed the Rollout plugin as outlined in the  [Rollout plugin installation guide](/docs/fleet-manager/rollout/rollout-plugin/).

## How to Perform a Unified Rollout

### Configuring the Rollout Policy

You can deploy a canary application demo using Kuma by the following command:

```console
kubectl apply -f examples/rollout/canaryKuma.yaml
```

Here is the configuration:

```yaml
apiVersion: apps.kurator.dev/v1alpha1
kind: Application
metadata:
  name: rollout-kuma-demo
  namespace: default
spec:
  source:
    gitRepository:
      interval: 3m0s
      ref:
        branch: master
      timeout: 1m0s
      url: https://github.com/stefanprodan/podinfo
  syncPolicies:
    - destination:
        fleet: quickstart
      kustomization:
        interval: 0s
        path: ./deploy/webapp
        prune: true
        timeout: 2m0s
      rollout:
        testLoader: true
        trafficRoutingProvider: kuma
        workload:
          apiVersion: apps/v1
          name: backend
          kind: Deployment
          namespace: webapp
        serviceName: backend
        port: 9898
        rolloutPolicy:
          trafficRouting:
            protocol: http
            timeoutSeconds: 60
            canaryStrategy:
              maxWeight: 50
              stepWeight: 10
          trafficAnalysis:
             checkIntervalSeconds: 90
             checkFailedTimes: 2
             metrics:
             - name: kuma-request-success-rate
               intervalSeconds: 90
               thresholdRange:
                 min: 99
               customMetric: 
                 provider: 
                   type: prometheus
                   address: http://prometheus-server.mesh-observability:80
                 query: |
                   sum(
                     rate(
                       http_requests_total{
                         status!~"5.*"
                       }[{{ interval }}]
                     )
                   )
                   /
                   sum(
                     rate(
                       http_requests_total[{{ interval }}]
                     )
                   ) * 100
             webhooks:
                 timeoutSeconds: 60
                 command:
                 - "hey -z 1m -q 10 -c 2 http://podinfo-canary.test:9898/"
          rolloutTimeoutSeconds: 600
    - destination:
        fleet: quickstart
      kustomization:
        targetNamespace: default
        interval: 5m0s
        path: ./kustomize
        prune: true
        timeout: 2m0s
```

**Notes:** There is a problem with the metric provided by the current flagger, so `customMetric` is used.Here is the detailed [API](/docs/references/app-api/#apps.kurator.dev/v1alpha1.Metric).

To use Kuma, you need to provide the `protocol` it uses. If you do not specify the protocol, Kurator will use `http` by default. 

Given the output provided, let's dive deeper to understand the various elements and their implications:

- Kurator allows customizing Rollout strategies under the `Spec.syncPolicies.rollout` section for services deployed via kustomization or helmrelease. It will establish and implement Canary Deployment for these services according to the configuration defined here.
- The `workload` defines the target resource for the Canary Deployment. The `kind` specifies the resource type, which can be either deployment or daemonset.
- The `serviceName` and `port` specify the name of the service for the workload as well as the exposed port number.
- The `trafficAnalysis` section defines the configuration for evaluating a new release version's health and readiness during a rollout process.
    - The `checkFailedTimes` parameter specifies the maximum number of failed check results allowed throughout the Canary Deployment lifecycle.
    - `checkIntervalSeconds` denotes the time interval between consecutive health evaluation checks.
    - The `metrics` identify the metrics that will be monitored to determine the deployment's health status. You can choose between the two built-in metric types `request-success-rate` and `request-duration` or write your own metric
    - The `webhooks` provide an extensibility mechanism for the analysis procedures. In this configuration, webhooks communicate with the testloader to generate test traffic for the healthchecks.
- The `trafficRouting` configuration specifies how traffic will be shifted to the canary deployment during the rollout process.
    - The `maxWeight` parameter defines the maximum percentage of traffic that can be routed to the canary before promotion.
    - `stepWeight` determines the incremental amount by which traffic will be increased after each successful analysis iteration, allowing the canary to be validated under a gradually growing proportion of real-world load. Kurator also supports configuring both the traffic settings for the full release after validation completes, as well as non-graduated traffic shifts during the testing period. Please refer to [Application API Reference](https://kurator.dev/docs/references/app-api/#apps.kurator.dev/v1alpha1.CanaryConfig) for more details on directly setting the release and test traffic distributions.
- The `rolloutStatus` section displays the actual processing status of rollout within the fleet.

About a minute after submitting this configuration, you can check the rollout status by running the following command:

```conole
kubectl get canary -n webapp --kubeconfig=/root/.kube/kurator-member1.config

NAME      STATUS        WEIGHT   LASTTRANSITIONTIME
backend   Initialized   0        2024-01-11T02:40:40Z
```

If the status shows as `Initialized`, it means the initialization of rollout process has completed successfully.

**Notes**: In the above configuration, we set the `kustomization.interval` to 0s. This disables Fluxcd's periodic synchronization of configurations between the local mirror and cluster. The reason is that Flagger needs to modify the replica counts in Deployments to complete its initialization process. If you are uncertain whether the replicas for all applications in your deployments are set to zero, it is recommended to also set `kustomization.interval` to 0s.

### Trigger Rollout

A Canary Deployment can be triggered by either updating the container image referenced in the git repository configuration, or directly updating the image of the deployment resource locally in the Kubernetes cluster.

Review the results:

```console
kubectl get canary -n webapp -w --kubeconfig=/root/.kube/kurator-member1.config

NAME      STATUS        WEIGHT   LASTTRANSITIONTIME
backend   Initialized   0        2024-01-11T02:40:40Z
backend   Progressing   0        2024-01-11T09:01:40Z
backend   Progressing   10       2024-01-11T09:03:10Z
backend   Progressing   10       2024-01-11T09:04:40Z
backend   Progressing   20       2024-01-11T09:06:10Z
backend   Progressing   30       2024-01-11T09:07:40Z
backend   Progressing   40       2024-01-11T09:09:10Z
backend   Progressing   50       2024-01-11T09:10:40Z
backend   Promoting     0        2024-01-11T09:12:10Z
backend   Finalising    0        2024-01-11T09:13:40Z
backend   Succeeded     0        2024-01-11T09:15:10Z
```

{{< image width="100%"
link="./image/canary.svg"
>}}

- As shown in the diagram, after triggering a canary deployment, the Kurator Rollout Plugin will first create pod(s) for the new version.
- It will then gradually shift traffic to the new version pod by increasing its traffic weight in the result metric over time. This `WEIGHT`  in the displayed result represents the current percentage of traffic accessing the new version pod during the analysis.
- Upon validating the new version through testing and confirming it is ready for release, Kurator will proceed to replace the old version with the new version across the entire cluster.
- It will then remove the canary pod, completing the rollout process.

```console
kubectl get application rollout-kuma-demo -oyaml

rolloutStatus:
  rolloutNameInCluster: backend
  rolloutStatusInCluster:
    canaryWeight: 0
    conditions:
    - lastTransitionTime: "2024-01-11T09:15:10Z"
      lastUpdateTime: "2024-01-11T09:15:10Z"
      message: Canary analysis completed successfully, promotion finished.
      reason: Succeeded
      status: "True"
      type: Promoted
    failedChecks: 1
    iterations: 0
    lastAppliedSpec: 7b779dcc48
    lastPromotedSpec: 7b779dcc48
    lastTransitionTime: "2024-01-11T09:15:10Z"
    phase: Succeeded
    trackedConfigs: {}
  clusterName: kurator-member1
```

A canary deployment is triggered by changes in any of the following objects:

- Deployment PodSpec (container image, command, ports, env, resources, etc)
- ConfigMaps mounted as volumes or mapped to environment variables
- Secrets mounted as volumes or mapped to environment variables

**Notes:** If you apply new changes to the deployment during the canary analysis, Kurator Rollout will restart the analysis.

## Cleanup

### 1.Cleanup the Rollout Policy

If you only need to remove the Rollout Policy, simply edit the current application and remove the corresponding description:

```console
kubectl edit application rollout-kuma-demo
```

To check the results of the deletion, you can observe that the rollout-related pods have been removed:

```console
kubectl get po -A --kubeconfig=/root/.kube/kurator-member1.config
kubectl get po -A --kubeconfig=/root/.kube/kurator-member2.config
```

If you want to configure a canary deployment for it again, you can simply edit the application and add the necessary configurations.

### 2.Cleanup the Application

When the application is delete, all associated resources will also be removed:

```console
kubectl delete application rollout-kuma-demo
```
