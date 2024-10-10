---
title: "Nginx A/B Testing"
linkTitle: "Nginx A/B Testing"
weight: 20
description: >
  A comprehensive guide on Kurator's A/B Testing uses Nginx as ingress, providing an overview and quick start guide.
---

## Prerequisites

In the subsequent sections, we'll guide you through a hands-on demonstration.

These are some of the prerequisites needed to use Kurator Rollout:

### Kubernetes Clusters

Kubernetes v1.27.3 or higher is supported.
  
You can use [Kind](https://kind.sigs.k8s.io/) to create clusters as needed.
It is recommended to use [Kurator's scripts](https://kurator.dev/docs/setup/install-cluster-operator/#setup-kubernetes-clusters-with-kind) to create multi-clusters environment.

Notes: You can find the mapping between Kind node image versions and Kubernetes versions on [Kind Release](https://github.com/kubernetes-sigs/kind/releases). Additionally, the website provides a lookup table showing compatible Kind and node image versions.


### Nginx

When Nginx is specified in fleet's `rollout.trafficRoutingProvider` , Kurator will install Nginx and its supporting Prometheus via helm in the fleet-managed clusters.

You can review the results a few minutes after applying fleet:

```console
kubectl get po -n ingress-nginx --kubeconfig=/root/.kube/kurator-member1.config   
NAME                                                              READY   STATUS    RESTARTS   AGE
ingress-nginx-flagger-kurator-member-7fbdfb7f7-hphc2              1/1     Running   0          5m44s
ingress-nginx-flagger-kurator-member-prometheus-56bdbf4855l4jkx   1/1     Running   0          5m44s
ingress-nginx-nginx-kurator-member-controller-6566b7886-b7g8f     1/1     Running   0          5m33s
ingress-nginx-testloader-kurator-member-loadtester-7ff7d75l2dwj   1/1     Running   0          5m51s

```

### Kurator Rollout Plugin

Before delving into the how to Perform a Unified Rollout, ensure you have successfully installed the Rollout plugin as outlined in the  [Rollout plugin installation guide](/docs/fleet-manager/rollout/rollout-plugin/).

## How to Perform a Unified Rollout

### Configuring the Rollout Policy

You can deploy a abtest application demo using Nginx by the following command:

```console
kubectl apply -f examples/rollout/ab-testingNginx.yaml
```

Here is the configuration:

```yaml
apiVersion: apps.kurator.dev/v1alpha1
kind: Application
metadata:
  name: abtesting-nginx-demo
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
        trafficRoutingProvider: nginx
        workload:
          apiVersion: apps/v1
          name: backend
          kind: Deployment
          namespace: webapp
        serviceName: backend
        port: 9898
        rolloutPolicy:
          trafficRouting:
            analysisTimes: 3
            timeoutSeconds: 60
            host: "app.example.com"
            match:
              - headers:
                  x-canary:
                    exact: "insider"
              - headers:
                  cookie:
                    exact: "canary"
          trafficAnalysis:
            checkIntervalSeconds: 90
            checkFailedTimes: 2
            metrics:
              - name: nginx-request-success-rate
                intervalSeconds: 90
                thresholdRange:
                  min: 99
                customMetric:
                  provider:
                    type: prometheus
                    address: http://ingress-nginx-flagger-kurator-member-prometheus.ingress-nginx:9090
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
                - "hey -z 1m -q 10 -c 2 http://app.example.com/"
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

To use Nginx, you need to provide the `host` it uses. Kurator will generate an ingress resource based on this field. Here is the [specific configuration generated](/docs/references/app-api/#apps.kurator.dev/v1alpha1.TrafficRoutingConfig). Other configurations are as follow.

Given the output provided, let's dive deeper to understand the various elements and their implications:

- Kurator allows customizing Rollout strategies under the `Spec.syncPolicies.rollout` section for services deployed via kustomization. It will establish and implement A/B Testing for these services according to the configuration defined here.
- The `workload` defines the target resource for the A/B Testing. The `kind` specifies the resource type, which can be either deployment or daemonset.
- The `serviceName` and `port` specify the name of the service for the workload as well as the exposed port number.
- The `trafficAnalysis` section defines the configuration for evaluating a new release version's health and readiness during a rollout process.
    - The `checkFailedTimes` parameter specifies the maximum number of failed check results allowed throughout the A/B Testing lifecycle.
    - `checkIntervalSeconds` denotes the time interval between consecutive health evaluation checks.
    - The `metrics` identify the metrics that will be monitored to determine the deployment's health status. You can choose between the two built-in metric types `request-success-rate` and `request-duration` or write your own metric
    - The `webhooks` provide an extensibility mechanism for the analysis procedures. In this configuration, webhooks communicate with the testloader to generate test traffic for the healthchecks.
- The `trafficRouting` configuration specifies how traffic will be shifted to the A/B Testing during the rollout process.
    - The `analysisTimes` signifies the number of testing iterations that will be conducted.
    - The `match` defines the criteria that an incoming request must satisfy in order to be routed to the new version. This includes header match definitions which specify rules for request headers. Other match dimensions like port, URL path etc. can also be configured. It's important to note that HTTP matching only takes effect during code analysis, and does not apply to normal usage afterwards. Please refer to [Application API Reference](https://kurator.dev/docs/references/app-api/#apps.kurator.dev/v1alpha1.TrafficRoutingConfig) for more details on directly setting the release and test traffic distributions.
- The `rolloutStatus` section displays the actual processing status of rollout within the fleet.

About a minute after submitting this configuration, you can check the rollout status by running the following command:

```console
kubectl get canary -n webapp --kubeconfig=/root/.kube/kurator-member1.config

NAME      STATUS        WEIGHT   LASTTRANSITIONTIME
backend   Initialized   0        2024-01-12T08:53:40Z
```

If the status shows as `Initialized`, it means the initialization of rollout process has completed successfully.

**Notes**: In the above configuration, we set the `kustomization.interval` to 0s. This disables Fluxcd's periodic synchronization of configurations between the local mirror and cluster. The reason is that Flagger needs to modify the replica counts in Deployments to complete its initialization process. If you are uncertain whether the replicas for all applications in your deployments are set to zero, it is recommended to also set `kustomization.interval` to 0s.

### Trigger Rollout

An A/B Testing can be triggered by either updating the container image referenced in the git repository configuration, or directly updating the image of the deployment resource locally in the Kubernetes cluster.

Review the results:

```console
kubectl get canary -n webapp -w --kubeconfig=/root/.kube/kurator-member1.config

NAME      STATUS        WEIGHT   LASTTRANSITIONTIME
backend   Initialized   0        2024-01-12T08:53:40Z
backend   Progressing   0        2024-01-12T08:55:10Z
backend   Progressing   0        2024-01-12T08:56:40Z
backend   Progressing   0        2024-01-12T08:58:10Z
backend   Progressing   0        2024-01-12T08:59:40Z
backend   Progressing   0        2024-01-12T09:01:10Z
backend   Promoting     0        2024-01-12T09:02:40Z
backend   Finalising    0        2024-01-12T09:04:10Z
backend   Succeeded     0        2024-01-12T09:05:40Z
```

{{< image width="100%"
link="./image/abtesting.svg"
>}}

- As shown in the diagram, after triggering an A/B Testing, the Kurator Rollout Plugin will first create pod(s) for the new version.
- The new version will then undergo multiple test iterations. During this testing period, incoming requests matching the defined criteria will be routed to the new version. Various testing metrics will be evaluated to determine the health and stability of the new release.
- Upon validating the new version through testing and confirming it is ready for release, Kurator will proceed to replace the old version with the new version across the entire cluster.
- It will then remove the canary pod, completing the rollout process.

```console
kubectl get application abtesting-nginx-demo -oyaml

rolloutStatus:
      backupNameInCluster: backend
      backupStatusInCluster:
        canaryWeight: 0
        conditions:
        - lastTransitionTime: "2024-01-12T09:05:40Z"
          lastUpdateTime: "2024-01-12T09:05:40Z"
          message: Canary analysis completed successfully, promotion finished.
          reason: Succeeded
          status: "True"
          type: Promoted
        failedChecks: 1
        iterations: 0
        lastAppliedSpec: 7b779dcc48
        lastPromotedSpec: 7b779dcc48
        lastTransitionTime: "2024-01-12T09:05:40Z"
        phase: Succeeded
        trackedConfigs: {}
      clusterName: kurator-member1
```

An A/B Testing is triggered by changes in any of the following objects:

- Deployment PodSpec (container image, command, ports, env, resources, etc)
- ConfigMaps mounted as volumes or mapped to environment variables
- Secrets mounted as volumes or mapped to environment variables

**Notes:** If you apply new changes to the deployment during the analysis, Kurator Rollout will restart the analysis.

## Cleanup

### 1.Cleanup the Rollout Policy

If you only need to remove the Rollout Policy, simply edit the current application and remove the corresponding description:

```console
kubectl edit application abtesting-nginx-demo
```

To check the results of the deletion, you can observe that the rollout-related pods have been removed:

```console
kubectl get po -A --kubeconfig=/root/.kube/kurator-member1.config
kubectl get po -A --kubeconfig=/root/.kube/kurator-member2.config
```

If you want to configure an A/B Testing for it again, you can simply edit the application and add the necessary configurations.

### 2.Cleanup the Application

When the application is delete, all associated resources will also be removed:

```console
kubectl delete application abtesting-nginx-demo
```