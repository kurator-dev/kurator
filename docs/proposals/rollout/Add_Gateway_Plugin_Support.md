---
title: Add gateway plugin support
authors:
- @Gidi233
reviewers:
approvers:

creation-date: 2024-07-23


---

## Add Gateway Plugin Support

### Summary

Kurator's release feature currently relies on the Istio gateway plugin to manage traffic distribution. To provide users with more options, we aim to extend Kurator's release feature to support additional common gateway plugins such as NGINX and Kuma.

### Motivation

By enhancing this feature, we can offer users more gateway options, simplify the necessary configurations, and reduce the learning curve.

#### Goals

1. Extend the gateway plugins supported by Kurator's release feature, initially including NGINX and Kuma.
2. Simplify user configuration and provide more options for traffic routing.

### Proposal

We propose adding support for NGINX and Kuma gateways by enhancing the fleet's reconciliation process to install these plugins based on the value of `rolloutPolicy.TrafficRoutingProvider`.

- If `rolloutPolicy.TrafficRoutingProvider == Nginx`, create an ingress according to the application configuration, wait for the ingress status to be complete, then update `Canary.Spec.IngressRef` and create the Canary.
- If `rolloutPolicy.TrafficRoutingProvider == Kuma`, create a namespace with the annotation `kuma.io/sidecar-injection=enabled`, add a protocol field to the application's API, and include `9898.service.kuma.io/protocol: protocol` in the annotations for the `apex, canary,  primary ` in `Canary.Spec.Service`.

### Design Details

We will delve into the API design required to support these configurations. The following is a preliminary design:

**Modification to `TrafficRoutingConfig` in the application:**

```go
type TrafficRoutingConfig struct {
	...
	// for NGINX
	// 默认创建的ingress如下,(replace app.example.com with your own domain,并根据需求更改路径匹配规则)
	// apiVersion: networking.k8s.io/v1
	// kind: Ingress
	// metadata:
	//   name: application.syncPolicies.rollout.name
	//   namespace: application.syncPolicies.rollout.namespace
	//   labels:
	//     app: application.syncPolicies.rollout.name
	//   annotations:
	//     kubernetes.io/ingress.class: "nginx"
	// spec:
	//   rules:
	//     - host: "app.example.com"
	//       http:
	//         paths:
	//           - pathType: Prefix
	//             path: "/"
	//             backend:
	//               service:
	//                 name: application.syncPolicies.rollout.name
	//                 port:
	//                   number: 80	
	Ingress ingressv1.IngressRule `json:"ingress,omitempty"`
	// for Kuma
	// Defaults to http
	Protocol string `json:"protocol,omitempty"`
}
```

**Modification to `FlaggerConfig` in the fleet:**

The `ProviderConfig` field is used for users to customize the selection of versions and configurations.

```go
type FlaggerConfig struct {
	...
	// ProviderConfig defines the configuration for the TrafficRoutingProvider.
	ProviderConfig   *Config   `json:"Config,omitempty"`
}

type Config struct {
	// Chart defines the helm chart config of the TrafficRoutingProvider.
	// default value is in ./pkg/fleet-manager/manifests/plugins/
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`
	// ExtraArgs is the set of extra arguments for TrafficRoutingProvider's chart.
	// You can pass in values according to your needs.
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}
```

**Default gateway configurations:**

*NGINX Configuration:*

```yaml
type: default
repo: https://kubernetes.github.io/ingress-nginx
name: nginx
version: 4.10.1
targetNamespace: ingress-nginx
values:
  controller:
    metrics:
      enabled: true
    podAnnotations:
      prometheus.io/scrape: true
      prometheus.io/port: 10254
```

*Kuma Configuration:*

```yaml
type: default
repo: https://kumahq.github.io/charts
name: kuma
version: 2.7.3
targetNamespace: kuma-system
values:
  controlPlane: 
    mode: zone 
```

We will merge these default configurations with the fleet's configuration, using the `plugin.tpl` template to generate the full helm configuration for deployment on each cluster.

#### Test Plan

During the development phase, we will add unit tests covering core functionalities and edge cases. Post-development, we will design integration tests to ensure proper rollout operations using Kuma and NGINX. Examples demonstrating rollouts with Kuma and NGINX will be provided.

