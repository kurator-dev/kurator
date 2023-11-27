---
title: Kurator's Continuous Delivery design 
authors:
- "@LiZhenCheng9527" # Authors' GitHub accounts here.
reviewers:
approvers:

creation-date: 2023-11-20

---

## Kurator's Continuous Delivery design

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

### Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. 

A good summary is probably at least a paragraph in length.
-->

Kurator, as an open-source distributed cloud-native platform, has been pivotal in aiding users to construct their distributed cloud-native infrastructure, thereby facilitating enterprise digital transformation.

To further enhance its functionality, this proposal designs Kurator's Continuous Delivery feature to meet users' needs for automated distribution of verified code.

By integrating Flagger, we aim to provide users with reliable, fast and unified Continuous Delivery, enabling them to easily distribute code across multiple clusters.

Base on Flagger, Kurator also offers A/B, Blue/Green and Canary distribution options. Meet the needs of the test verification release.

### Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users.
-->

With the increase in project size and complexity and the development of cloud computing technology, the CI/CD process has been proposed.

CI/CD has many advantages such as increased development efficiency, improved quality, more reliable deployment and better continuous learning and improvement, which is more suitable for today's software development process.

Kurator is an open source distributed cloud native suite that provides users with a one-stop open source solution for distributed cloud native scenarios.

Therefore, CI/CD as an important feature of cloud native usage scenarios, Kurator needs to provide relevant functional support to achieve the vision of Kurator unified configuration distribution.

#### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

Unified configuration distribution only requires the user to declare the desired API state in one place, and Kurator will automatically handle all subsequent operations.

Kurator deploys different daemons on different nodes in the cluster with different rules for configuration distribution based on the declarations in the Spec.

In Kurator, you can choose to distribute the application with the same configuration to a specific single or multiple clusters for verification.

- **unified Continuous Delivery**
    - Supports unified configuration of releases for multiple clusters. Achieve the deployment configuration of the application to be distributed to the specified single or multiple clusters.
    - Support A/B, Blue/Green, Canary three release validation rules configuration.
    - Supports automatic rollback when release validation fails.

#### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- **Traffic distribution tools that Flagger can use other than istio are not supported** While Flagger is able to support a wide range of traffic distribution tools including istio, nginx for grey scale releases. However, Kuraotr currently only supports unified installation of istio in multiple clusters, and Kurator may implement unified installation of other traffic distribution tools in the future.

### Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->
The purpose of this proposal is to introduce a unified Continuous Delivery for Kurator that supports A/B, Blue/Green, and Canary.The main objectives of this proposal are as follows:

Custom Resource Definitions (CRDs): Design CRDs to enable Uniform Continuous Delivery These CRDs will provide a structured approach to defining clusters and different configuration distribution rules to enable uniform configuration distribution.

Fleet-Manager Implementation: The Cluster Manager component will be responsible for monitoring the CRDs and performing the defined functions. It will install Flagger on the clusters and handle potential errors or anomalies to ensure smooth operation.

By integrating these enhancements, Kurator will provide users with a powerful and streamlined solution for managing the task of implementing Unified Configuration Distribution and simplifying the overall operational process.

#### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

##### Story 1

**User Role**: Cloud Native Project Development Team.

**Feature**: With the enhanced Kurator, developers can easily deploy their new releases to multiple clusters for validation testing.

**Value**: Provides a simplified, automated way to unify the management of configuration distribution across multiple clusters. Reduces human error and ensures data continuity and compliance.

**Outcome**: With this feature, developers can easily assign uniform configurations to multiple clusters to improve reliability, availability, and storage efficiency for business publishing and easily achieve scalability.

##### Story 2

**User Role**: Enterprise Product Development Project Team.

**Feature**: With the enhanced Kurator, developers can quickly release and A/B, Blue/Green or Canary test new requirements in their environment after they are completed.

**Value**: Provides a simplified, automated way for developers to distribute configurations in a uniform manner. Enables validation testing in multiple usage environments. Provides A/B, Blue/Green or Canary tests to meet different testing needs.

**Outcome**: With this feature, developers can easily assign uniform configurations to multiple clusters, test new releases, and ensure the quality of new releases. In addition, it also provides automatic rollback function when the test fails, reducing the developer's operational burden and bug impact time.

#### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

#### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? 

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

### Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

In this section, we'll dive into the detailed API design for the Unified Continuous Delivery Feature.

These APIs are designed to facilitate Kurator's integration with Flagger to enable the required functionality.

Unlike Flagger, we may need to adjust Unified Continuous Delivery to reflect our new strategy and decisions.

#### Unified Continuous Delivery API

Kurator is designed to unify the installation of Flagger as a fleet plugin in a given single or multiple clusters.

Then use the kurator application to distribute the Flagger configuration. Kurator's unified configuration distribution.

Kurator puts the Continuous Delivery's api under the app's api, so that when Kurator deploys the app in the target cluster, it also deploys the corresponding Continuous Delivery policy.

Here's the preliminary design for the Unified Continuous Delivery:

```console
// ApplicationSyncPolicy defines the configuration to sync an artifact.
// Only oneof `kustomization` or `helm` can be specified to manage application sync.
// ApplicationSyncPolicy distributes the Continuous Delivery configuration 
// at the same time as the application deployment, if needed. 
type ApplicationSyncPolicy struct {
    // Name defines the name of the sync policy.
    // If unspecified, a name of format `<application name>-<index>` will be generated.
    // +optional
    Name string `json:"name,omitempty"`

    // Kustomization defines the configuration to calculate the desired state
    // from a source using kustomize.
    // +optional
    Kustomization *Kustomization `json:"kustomization,omitempty"`
    // HelmRelease defines the desired state of a Helm release.
    // +optional
    Helm *HelmRelease `json:"helm,omitempty"`

    // Destination defines the destination for the artifact.
    // If specified, it will override the destination specified at Application level.
    // +optional
    Destination *ApplicationDestination `json:"destination"`

    // Delivery defines the Continuous Delivery Configurations to be used.
    // If specified, a uniform Continuous Delivery policy is configured for this installed object.
    // +optional
    Delivery *DeliveryConfig `json:"deliveryPolicy"`
}

type DeliveryConfig struct {
    // Autoscale defines minreplicas and maxreplicas of horizontal Pod Autoscaler.
    // +optional
    Autoscale *AutoscaleRef `json:"autoScale,omitempty"`

    // Whether or not the testload is installed by the user. The default is false.
    // If set it to ture, user need to install the testload himself.
    // If set it to false or leave it blank, Kurator will install the flagger's testload.
    // +optional
    TestLoad bool `json:"testLoad"`

    // DeliveryPolicy defines the Release Strategy of Object
    DeliveryPolicy *DeliveryPolicy `json:"deliveryPolicy"`
}
```

Kurator generates horizontalpodautoscaler.autoscaling resources based on `autoscale` configurations as stable and to-be-verified versions. For instance tuning during testing

```concle
type AutoscaleRef struct {
    // +optional
    MinReplicas *int32 `json:"minReplicas,omitempty"`

    // +optional
    MaxReplicas *int32 `json:"maxReplicas,omitempty"`
}
```

`Testload` indicates whether the user wants to install the test traffic load themselves. If you don't want to install the testload yourself, Kurator will install flagger's testload by default.

`DeliveryPolicy` defines the Continuous Delivery configuration for this installation object. Although in Kurator is not detailed into the canary release, A/B test and blue-green test. However, users can freely configure their own traffic validation policies through the DeliveryPolicy. Complete the release test.

```console
// Note: refer to https://github.com/fluxcd/flagger/blob/main/pkg/apis/flagger/v1beta1/canary.go
type DeliveryPolicy struct {
    // Kurator only supports istio for now.
    // New Provider will be added later.
    // +optional
    TrafficRoutingProvider string `json:"trafficRoutingProvider,omitempty"`

    // TargetObject specifies what object to deploy the test to. 
    // Objects of type deployment or daemonSet.
    TargetObject *TargetObjectReference `json:"targetObject"`

    // ServiceCfg is the configuration of the service pointing to the TargetObject.
    ServiceCfg *ServiceConfig `json:"serviceCfg"`

    // The virtualserviceCfg defines the configuration of the gateway, traffic distribution rules, and so on.
    VirtualServiceCfg *VirtualServiceConfig `json:"virtualServiceCfg"`

    // TrafficAnalysis defines the validation process of a release
    TrafficAnalysis *TrafficAnalysis `json:"trafficAnalysis,omitempty"`

    // ProgressDeadlineSeconds represents the maximum time in seconds for a
    // canary deployment to make progress before it is considered to be failed.
    // Defaults to 600s.
    // +optional
    ProgressDeadlineSeconds *int32 `json:"progressDeadlineSeconds,omitempty"`

    // SkipAnalysis promotes the canary without analysing it
    // +optional
    SkipTrafficAnalysis bool `json:"skipTrafficAnalysis,omitempty"`

    // Restore resources to initial state when deleting canary resources.
    // Use of the revertOnDeletion property should be enabled
    // when you no longer plan to rely on Kurator for deployment management.
    // +optional
    RevertOnDeletion bool `json:"revertOnDeletion,omitempty"`

    // Suspend, if set to true will suspend the Canary, disabling any canary runs
    // regardless of any changes to its target, services, etc. Note that if the
    // Canary is suspended during an analysis, its paused until the Canary is unsuspended.
    // +optional
    Suspend bool `json:"suspend,omitempty"`
}
```

TargetObjectReference contains enough information to let you locate the typed referenced object in the same namespace. The two types of Kind now supported are `Deployment` and `DaemonSet`.

```concole
// Note: refer to https://github.com/fluxcd/flagger/blob/main/pkg/apis/flagger/v1beta1/canary.go
type TargetObjectReference struct {
    // API version of the referent
    // +optional
    APIVersion string `json:"apiVersion,omitempty"`

    // Kind of the referent
    // +optional
    Kind string `json:"kind,omitempty"`

    // Name of the referent
    Name string `json:"name"`
}
```

The ServiceConfig is the configuration of the service that is generated according to the object specified in the TargetObject. The main thing is to specify the `Port` and `targetPort`.

```console
// Note: refer to https://github.com/fluxcd/flagger/blob/main/pkg/apis/flagger/v1beta1/canary.go
type ServiceConfig struct {
    // Name of the virtual Kubernetes generated by Flagger.
    // Defaults to DeliveryPolicy.TargetObject.Name
    // +optional
    Name string `json:"name,omitempty"`

    // Port of the generated Kubernetes service.
    Port int32 `json:"port"`

    // Port name of the generated Kubernetes service.
    // Defaults to http
    // +optional
    PortName string `json:"portName,omitempty"`

    // Target port number or name of the generated Kubernetes service.
    // Defaults to CanaryService.Port
    // +optional
    TargetPort intstr.IntOrString `json:"targetPort,omitempty"`

    // AppProtocol of the service.
    // https://kubernetes.io/docs/concepts/services-networking/service/#application-protocol
    // +optional
    AppProtocol string `json:"appProtocol,omitempty"`

    // PortDiscovery adds all container ports to the generated Kubernetes service.
    // Defaults to true
    PortDiscovery bool `json:"portDiscovery"`
}
```

Kurator will create a VirtualService resource based on the configuration in `VirtualServiceConfig` to distribute traffic.

```console
// Note: refer to https://github.com/fluxcd/flagger/blob/main/pkg/apis/flagger/v1beta1/canary.go
type VirtualServiceConfig struct {
    // Timeout of the HTTP or gRPC request.
    // +optional
    Timeout string `json:"timeout,omitempty"`

    // Gateways attached to the generated Istio virtual service.
    // Defaults to the internal mesh gateway.
    // +optional
    Gateways []string `json:"gateways,omitempty"`

    // Hosts attached to the generated Istio virtual service.
    // Defaults to the service name
    // +optional
    Hosts []string `json:"hosts,omitempty"`

    // If enabled, Flagger would generate Istio VirtualServices without hosts and gateway,
    // making the service compatible with Istio delegation.
    // If delegation is enabled, Flagger makes the service compatible with Istio delegation,
    // enabling configuration of traffic within the cluster
    // Note that pilot env `PILOT_ENABLE_VIRTUAL_SERVICE_DELEGATE` must also be set.
    // +optional
    Delegation bool `json:"delegation,omitempty"`

    // TrafficPolicy attached to the generated Istio destination rules.
    // +optional
    TrafficPolicy *istiov1alpha3.TrafficPolicy `json:"trafficPolicy,omitempty"`

    // URI match conditions for the generated service.
    // +optional
    Match []istiov1alpha3.HTTPMatchRequest `json:"match,omitempty"`

    // Retries policy for the generated virtual service.
    // +optional
    Retries *istiov1alpha3.HTTPRetry `json:"retries,omitempty"`

    // Headers operations for the generated Istio virtual service.
    // e.g.
    // headers:
    //   request:
    //     add:
    //       x-some-header: "value"
    // +optional
    Headers *istiov1alpha3.Headers `json:"headers,omitempty"`

    // Cross-Origin Resource Sharing policy for the generated Istio virtual service.
    // e.g.
    // corsPolicy:
    //   allowHeaders:
    //   - x-some-header
    //   allowMethods:
    //   - GET
    //   allowOrigin:
    //   - example.com
    //   maxAge: 24h
    // +optional
    CorsPolicy *istiov1alpha3.CorsPolicy `json:"corsPolicy,omitempty"`

    // Primary is the metadata to add to the primary service.
    // +optional
    Primary *CustomMetadata `json:"primary,omitempty"`

    // Canary is the metadata to add to the canary service.
    // +optional
    Canary *CustomMetadata `json:"canary,omitempty"`
}
```

As part of the TrafficAnalysis process, Kurator can validate service level objectives (SLOs) like availability, error rate percentage, average response time and any other objective based on app specific metrics. If a drop in performance is noticed during the SLOs analysis, the release will be automatically rolled back with minimum impact to end-users.

```concole
// Note: refer to https://github.com/fluxcd/flagger/blob/main/pkg/apis/flagger/v1beta1/canary.go
type TrafficAnalysis struct {
    // Schedule interval for this traffic analysis
    Interval string `json:"interval"`

    // Number of checks to run for A/B Testing and Blue/Green
    // +optional
    Iterations int `json:"iterations,omitempty"`

    // Enable traffic mirroring for Blue/Green
    // +optional
    Mirror bool `json:"mirror,omitempty"`

    // Weight of the traffic to be mirrored in the range of [0, 100].
    // +optional
    MirrorWeight int `json:"mirrorWeight,omitempty"`

    // Max traffic weight routed to canary test
    // +optional
    MaxWeight int `json:"maxWeight,omitempty"`

    // Incremental traffic weight step for analysis phase
    // +optional
    StepWeight int `json:"stepWeight,omitempty"`

    // Incremental traffic weight steps for analysis phase
    // +optional
    StepWeights []int `json:"stepWeights,omitempty"`

    // Incremental traffic weight step for promotion phase
    // +optional
    StepWeightPromotion int `json:"stepWeightPromotion,omitempty"`

    // Max number of failed checks before the traffic analysis is terminated
    Threshold int `json:"threshold"`

    // Percentage of pods that need to be available to consider primary as ready
    // Defaults to 100
    PrimaryReadyThreshold *int `json:"primaryReadyThreshold,omitempty"`

    // Percentage of pods that need to be available to consider canary as ready
    // Defaults to 100
    CanaryReadyThreshold *int `json:"canaryReadyThreshold,omitempty"`

    // Metric check list for this traffic analysis
    // Flagger comes with two builtin metric checks: HTTP request success rate and duration.
    // Can use either built-in metric checks or custom checks.
    // If you want use custom checks, you can refer to https://docs.flagger.app/usage/metrics#custom-metrics.
    // +optional
    Metrics []Metric `json:"metrics,omitempty"`

    // Webhook list for this traffic  analysis
    // +optional
    Webhooks []Webhook `json:"webhooks,omitempty"`

    // A/B testing HTTP header match conditions
    // +optional
    Match []istiov1alpha3.HTTPMatchRequest `json:"match,omitempty"`

    // SessionAffinity represents the session affinity settings for a analysis run.
    // +optional
    SessionAffinity *SessionAffinity `json:"sessionAffinity,omitempty"`
}

type Metric struct {
    // Name of the metric
    // The name of the two builtin metric checks is "request-success-rate" and "request-duration".
    // User also can use Name point to custom metric checks
    Name string `json:"name"`

    // Metrics query interval
    Interval string `json:"interval,omitempty"`

    // Range value accepted for this metric
    // +optional
    ThresholdRange *CanaryThresholdRange `json:"thresholdRange,omitempty"`

    // TemplateRef references a metric template object
    // +optional
    TemplateRef *CrossNamespaceObjectReference `json:"templateRef,omitempty"`

    // TemplateVariables provides a map of key/value pairs that can be used to inject variables into a metric query.
    // +optional
    TemplateVariables map[string]string `json:"templateVariables,omitempty"`
}

// CanaryThresholdRange defines the range used for metrics validation
type CanaryThresholdRange struct {
    // Minimum value
    // +optional
    Min *float64 `json:"min,omitempty"`

    // Maximum value
    // +optional
    Max *float64 `json:"max,omitempty"`
}

// CrossNamespaceObjectReference contains enough information to let you locate the
// typed referenced object at cluster level
type CrossNamespaceObjectReference struct {
    // API version of the referent
    // +optional
    APIVersion string `json:"apiVersion,omitempty"`

    // Kind of the referent
    // +optional
    Kind string `json:"kind,omitempty"`

    // Name of the referent
    Name string `json:"name"`

    // Namespace of the referent
    // +optional
    Namespace string `json:"namespace,omitempty"`
}

// The traffic analysis can be extended with webhooks.
// Kurator will call each webhook URL and determine from the response status code (HTTP 2xx)
// if the test is failing or not.
// e.g.
// webhooks:
//   - name: "start gate"
//     type: confirm-rollout
//     url: http://flagger-loadtester.test/gate/approve
type Webhook struct {
    // Type of this webhook
    // Different types mean different actions when the webhook check fails.
    Type HookType `json:"type"`

    // Name of this webhook
    Name string `json:"name"`

    // URL address of this webhook
    URL string `json:"url"`

    // Request timeout for this webhook
    Timeout string `json:"timeout,omitempty"`

    // Metadata (key-value pairs) for this webhook
    // +optional
    Metadata *map[string]string `json:"metadata,omitempty"`
}

// HookType can be pre, post or during rollout
type HookType string

const (
    // RolloutHook execute webhook during the canary analysis
    RolloutHook HookType = "rollout"
    // PreRolloutHook execute webhook before routing traffic to canary
    PreRolloutHook HookType = "pre-rollout"
    // PostRolloutHook execute webhook after the canary analysis
    PostRolloutHook HookType = "post-rollout"
    // ConfirmRolloutHook halt canary analysis until webhook returns HTTP 200
    ConfirmRolloutHook HookType = "confirm-rollout"
    // ConfirmPromotionHook halt canary promotion until webhook returns HTTP 200
    ConfirmPromotionHook HookType = "confirm-promotion"
    // EventHook dispatches Flagger events to the specified endpoint
    EventHook HookType = "event"
    // RollbackHook rollback canary analysis if webhook returns HTTP 200
    RollbackHook HookType = "rollback"
    // ConfirmTrafficIncreaseHook increases traffic weight if webhook returns HTTP 200
    ConfirmTrafficIncreaseHook = "confirm-traffic-increase"
)

type SessionAffinity struct {
    // CookieName is the key that will be used for the session affinity cookie.
    CookieName string `json:"cookieName,omitempty"`
    // MaxAge indicates the number of seconds until the session affinity cookie will expire.
    // ref: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie#attributes
    // The default value is 86,400 seconds, i.e. a day.
    // +optional
    MaxAge int `json:"maxAge,omitempty"`
}
```

#### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

-->

End-to-End Tests: Comprehensive E2E tests should be conducted to ensure the  Continuous Delivery processes work seamlessly across different clusters.

Integration Tests: Integration tests should be designed to ensure Kurator's integration with Flagger functions as expected.

Unit Tests: Unit tests should cover the core functionalities and edge cases.

Isolation Testing: The Delivery processes functionalities should be tested in isolation and in conjunction with other components to ensure compatibility and performance.

### Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

<!--
Note: This is a simplified version of kubernetes enhancement proposal template.
https://github.com/kubernetes/enhancements/tree/3317d4cb548c396a430d1c1ac6625226018adf6a/keps/NNNN-kep-template
-->

Alternative: Integrating with Other CD Tools

Consideration: Integrating with other existing Continuous Delivery tools like Argo CD, Argo Rollout was also considered.

Rationale for Rejection: While these tools are powerful, they may not offer the same level of customization and Kubernetes-native capabilities as Flagger.
Additionally, this approach would have required extensive modifications to align with the cloud-native focus of the Kurator project.
