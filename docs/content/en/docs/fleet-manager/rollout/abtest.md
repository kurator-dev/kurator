---
title: "A/B Testing"
linkTitle: "A/B Testing"
weight: 30
description: >
  A comprehensive guide on Kurator's A/B Testing, providing an overview and quick start guide.
---

## Introduction

A/B Testing is a method of comparing two versions of an application to validate which performs better.
It essentially involves a controlled experiment where users are randomly allocated into groups at the same time, with each group experiencing a different version of the application.
The metrics from their usage are then analyzed to select the superior version based on the results. The A/B Testing can also be used to route selective users to the new version, allowing their real-world feedback on the new release to be gathered.

- **Use Case**: There are two application services with identical backend functionality but different frontend UIs. It is now necessary to validate which UI design leads to a better user experience. In this scenario, A/B Testing should be used to deploy both versions of the service in a live environment. The UI that demonstrates superior user metrics and outcomes can then be selected for full release.
- **Functionality**: Provide configuration of A/B Testing and trigger an A/B Testing on new release.

By allowing users to deploy applications and their A/B Testing configurations in a single place, Kurator streamlines A/B Testing through automated GitOps workflows for unified deployment and validation.

## Prerequisites

In the subsequent sections, we'll guide you through a hands-on demonstration.

These are some of the prerequisites needed to use Kurator Rollout:

### Kubernetes Clusters

Kubernetes v1.27.3 or higher is supported.
  
You can use [Kind](https://kind.sigs.k8s.io/) to create clusters as needed.
It is recommended to use [Kurator's scripts](https://kurator.dev/docs/setup/install-cluster-operator/#setup-kubernetes-clusters-with-kind) to create multi-clusters environment.

Notes: You can find the mapping between Kind node image versions and Kubernetes versions on [Kind Release](https://github.com/kubernetes-sigs/kind/releases). Additionally, the website provides a lookup table showing compatible Kind and node image versions.

### Istio

Istio v1.18 or higher is supported.

It is recommended to install Istio using istioctl. Refer to [istio documentation](https://istio.io/latest/docs/ops/diagnostic-tools/istioctl/) for instructions on installing istioctl.

After installing istioctl, you can install Istio using the following commands:

```console
export KUBECONFIG=/root/.kube/kurator-member1.config

istioctl manifest install --set profile=default
```

Review the results:

```console
kubeclt get po -n istio-system --kubeconfig=/root/.kube/kurator-member1.config

istio-system         istio-ingressgateway-65b5c9f9bb-8w5ml                   1/1     Running   0          37s
istio-system         istiod-657f7686cf-hshwp                                 1/1     Running   0          2m5s
```

### Prometheus

Kurator needs to use Prometheus to collect metrics on monitored resources in the cluster, which will be used as the basis for determining whether to continue rolling out.

You can install Prometheus using the following commands:

```console
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.20/samples/addons/prometheus.yaml
```

Review the results:

```console
Kubectl get po -n istio-system --kubeconfig=/root/.kube/kurator-member1.config

istio-system         prometheus-5d5d6d6fc-5hxbh                              2/2     Running   0          91s
```

Notes: It is recommended to change the release in the above commands to match the version of Istio that was already installed.

### Ingress Gateway

To allow external programs access to the services you've deployed, you will need to create an IngressGateway resource.

You can create an ingress gateway using the following commands:

```console
kubectl apply -f -<<EOF
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: public-gateway
  namespace: istio-system
spec:
  selector:
    istio: ingressgateway
  servers:
    - port:
        number: 80
        name: http
        protocol: HTTP
      hosts:
        - "*"
EOF
```

Review the results:

```console
Kubectl get gateway -n istio-system --kubeconfig=/root/.kube/kurator-member1.config

istio-system   public-gateway   17s
```

### Kurator Rollout Plugin

Before delving into the how to Perform a Unified Rollout, ensure you have successfully installed the Rollout plugin as outlined in the  [Rollout plugin installation guide](rollout-plugin.md).

## How to Perform a Unified Rollout

### Configuring the Rollout Policy

You can initiate the process by deploying a application demo using the following command:

```console
kubectl apply -f examples/rollout/ab-testing.yaml
```

Review the results:

```console
kubectl get application abtesting-demo -oyaml
```

The expected result should be:

```console
apiVersion: apps.kurator.dev/v1alpha1
kind: Application
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps.kurator.dev/v1alpha1","kind":"Application","metadata":{"annotations":{},"name":"abtesting-demo","namespace":"default"},"spec":{"source":{"gitRepository":{"interval":"25m0s","ref":{"branch":"master"},"timeout":"1m0s","url":"https://github.com/stefanprodan/podinfo"}},"syncPolicies":[{"destination":{"fleet":"quickstart"},"kustomization":{"interval":"0s","path":"./deploy/webapp","prune":true,"timeout":"2m0s"},"rollout":{"port":9898,"rolloutPolicy":{"rolloutTimeoutSeconds":600,"trafficAnalysis":{"checkFailedTimes":2,"checkIntervalSeconds":90,"metrics":[{"intervalSeconds":90,"name":"request-success-rate","thresholdRange":{"min":99}},{"intervalSeconds":90,"name":"request-duration","thresholdRange":{"max":500}}],"webhooks":{"command":["hey -z 1m -q 10 -c 2 http://backend-canary.webapp:9898/"],"timeoutSeconds":60}},"trafficRouting":{"analysisTimes":3,"gateways":["istio-system/public-gateway"],"hosts":["backend.webapp"],"match":[{"headers":{"user-agent":{"regex":".*Firefox.*"}}},{"headers":{"cookie":{"regex":"^(.*?;)?(type=insider)(;.*)?$"}}}],"timeoutSeconds":60}},"serviceName":"backend","testLoader":true,"trafficRoutingProvider":"istio","workload":{"apiVersion":"apps/v1","kind":"Deployment","name":"backend","namespace":"webapp"}}},{"destination":{"fleet":"quickstart"},"kustomization":{"interval":"5m0s","path":"./kustomize","prune":true,"targetNamespace":"default","timeout":"2m0s"}}]}}
  creationTimestamp: "2024-01-12T08:52:05Z"
  finalizers:
  - apps.kurator.dev
  generation: 1
  name: abtesting-demo
  namespace: default
  resourceVersion: "580416"
  uid: 5cc19ce4-2a47-45b3-be6d-ce8ca1ab6a63
spec:
  source:
    gitRepository:
      gitImplementation: go-git
      interval: 25m0s
      ref:
        branch: master
      timeout: 1m0s
      url: https://github.com/stefanprodan/podinfo
  syncPolicies:
  - destination:
      fleet: quickstart
    kustomization:
      force: false
      interval: 0s
      path: ./deploy/webapp
      prune: true
      timeout: 2m0s
    rollout:
      port: 9898
      rolloutPolicy:
        rolloutTimeoutSeconds: 600
        trafficAnalysis:
          checkFailedTimes: 2
          checkIntervalSeconds: 90
          metrics:
          - intervalSeconds: 90
            name: request-success-rate
            thresholdRange:
              min: 99
          - intervalSeconds: 90
            name: request-duration
            thresholdRange:
              max: 500
          webhooks:
            command:
            - hey -z 1m -q 10 -c 2 http://backend-canary.webapp:9898/
            timeoutSeconds: 60
        trafficRouting:
          analysisTimes: 3
          gateways:
          - istio-system/public-gateway
          hosts:
          - backend.webapp
          match:
          - headers:
              user-agent:
                regex: .*Firefox.*
          - headers:
              cookie:
                regex: ^(.*?;)?(type=insider)(;.*)?$
          timeoutSeconds: 60
      serviceName: backend
      testLoader: true
      trafficRoutingProvider: istio
      workload:
        apiVersion: apps/v1
        kind: Deployment
        name: backend
        namespace: webapp
  - destination:
      fleet: quickstart
    kustomization:
      force: false
      interval: 5m0s
      path: ./kustomize
      prune: true
      targetNamespace: default
      timeout: 2m0s
status:
  sourceStatus:
    gitRepoStatus:
      artifact:
        digest: sha256:8d86ecbdb528263637786ff0ad07491b4f78781626695ffa8bd9649032699636
        lastUpdateTime: "2024-01-12T08:52:07Z"
        path: gitrepository/default/abtesting-demo/dc830d02a6e0bcbf63bcc387e8bde57d5627aec2.tar.gz
        revision: master@sha1:dc830d02a6e0bcbf63bcc387e8bde57d5627aec2
        size: 282288
        url: http://source-controller.fluxcd-system.svc.cluster.local./gitrepository/default/abtesting-demo/dc830d02a6e0bcbf63bcc387e8bde57d5627aec2.tar.gz
      conditions:
      - lastTransitionTime: "2024-01-12T08:52:07Z"
        message: stored artifact for revision 'master@sha1:dc830d02a6e0bcbf63bcc387e8bde57d5627aec2'
        observedGeneration: 1
        reason: Succeeded
        status: "True"
        type: Ready
      - lastTransitionTime: "2024-01-12T08:52:07Z"
        message: stored artifact for revision 'master@sha1:dc830d02a6e0bcbf63bcc387e8bde57d5627aec2'
        observedGeneration: 1
        reason: Succeeded
        status: "True"
        type: ArtifactInStorage
      observedGeneration: 1
      url: http://source-controller.fluxcd-system.svc.cluster.local./gitrepository/default/abtesting-demo/latest.tar.gz
  syncStatus:
  - kustomizationStatus:
      conditions:
      - lastTransitionTime: "2024-01-13T06:47:52Z"
        message: 'Applied revision: master@sha1:dc830d02a6e0bcbf63bcc387e8bde57d5627aec2'
        observedGeneration: 1
        reason: ReconciliationSucceeded
        status: "True"
        type: Ready
      inventory:
        entries:
        - id: default_podinfo__Service
          v: v1
        - id: default_podinfo_apps_Deployment
          v: v1
        - id: default_podinfo_autoscaling_HorizontalPodAutoscaler
          v: v2
      lastAppliedRevision: master@sha1:dc830d02a6e0bcbf63bcc387e8bde57d5627aec2
      lastAttemptedRevision: master@sha1:dc830d02a6e0bcbf63bcc387e8bde57d5627aec2
      observedGeneration: 1
    name: abtesting-demo-0-attachedcluster-kurator-member1
    rolloutStatus:
      backupNameInCluster: backend
      backupStatusInCluster:
        canaryWeight: 0
        conditions:
        - lastTransitionTime: "2024-01-12T09:05:40Z"
          lastUpdateTime: "2024-01-12T09:05:40Z"
          message: Deployment initialization completed.
          reason: Initialized
          status: "True"
          type: Promoted
        failedChecks: 0
        iterations: 0
        lastAppliedSpec: 7b779dcc48
        lastPromotedSpec: 7b779dcc48
        lastTransitionTime: "2024-01-12T09:05:40Z"
        phase: Initialized
        trackedConfigs: {}
      clusterName: kurator-member1
  - kustomizationStatus:
      conditions:
      - lastTransitionTime: "2024-01-13T06:47:52Z"
        message: 'Applied revision: master@sha1:dc830d02a6e0bcbf63bcc387e8bde57d5627aec2'
        observedGeneration: 1
        reason: ReconciliationSucceeded
        status: "True"
        type: Ready
      inventory:
        entries:
        - id: default_podinfo__Service
          v: v1
        - id: default_podinfo_apps_Deployment
          v: v1
        - id: default_podinfo_autoscaling_HorizontalPodAutoscaler
          v: v2
      lastAppliedRevision: master@sha1:dc830d02a6e0bcbf63bcc387e8bde57d5627aec2
      lastAttemptedRevision: master@sha1:dc830d02a6e0bcbf63bcc387e8bde57d5627aec2
      observedGeneration: 1
    name: abtesting-demo-1-attachedcluster-kurator-member1
```

Given the output provided, let's dive deeper to understand the various elements and their implications:

- Kurator allows customizing Rollout strategies under the `Spec.syncPolicies.rollout` section for services deployed via kustomization. It will establish and implement A/B Testing for these services according to the configuration defined here.
- The `workload` defines the target resource for the A/B Testing. The `kind` specifies the resource type, which can be either deployment or daemonset.
- The `serviceName` and `port` specify the name of the service for the workload as well as the exposed port number.
- The `trafficAnalysis` section defines the configuration for evaluating a new release version's health and readiness during a rollout process.
    - The `checkFailedTimes` parameter specifies the maximum number of failed check results allowed throughout the A/B Testing lifecycle.
    - `checkIntervalSeconds` denotes the time interval between consecutive health evaluation checks.
    - The `metrics` identify the metrics that will be monitored to determine the deployment's health status. Currently, only `request-success-rate` and `request-duration` two built-in metric types are supported.
    - The `webhooks` provide an extensibility mechanism for the analysis procedures. In this configuration, webhooks communicate with the testloader to generate test traffic for the healthchecks.
- The `trafficRouting` configuration specifies how traffic will be shifted to the A/B Testing during the rollout process.
    - The `analysisTimes` signifies the number of testing iterations that will be conducted.
    - The `match` defines the criteria that an incoming request must satisfy in order to be routed to the new version. This includes header match definitions which specify rules for request headers. Other match dimensions like port, URL path etc. can also be configured. It's important to note that HTTP matching only takes effect during code analysis, and does not apply to normal usage afterwards. Please refer to [Application API Reference](https://kurator.dev/docs/references/app-api/#apps.kurator.dev/v1alpha1.TrafficRoutingConfig) for more details on directly setting the release and test traffic distributions.
    - The `gateways` and `host` represent the ingress points for external and internal service traffic, respectively.
- The `rolloutStatus` section displays the actual processing status of rollout within the fleet.

About a minute after submitting this configuration, you can check the rollout status by running the following command:

```conole
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
kubectl get application rolllout-demo -oyaml

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
kubectl edit applicaiton abtesting-demo
```

To check the results of the deletion, you can observe that the rollout-related pods have been removed:

```console
kubectl get po -A --kubeconfig=/root/.kube/kurator-member1.config
kubectl get po -A --kubeconfig=/root/.kube/kurator-member2.config
```

If you want to configure an A/B Testing for it again, you can simply edit the application and add the necessary configurations.

### 2.Cleanup the Application

When the application is delete, all associated resources will also be reomved:

```console
kubectl delete application abtesting-demo
```
