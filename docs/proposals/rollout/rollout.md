---
title: Kurator's Rollout design 
authors:
- @LiZhenCheng9527 # Authors' GitHub accounts here.
reviewers:
approvers:

creation-date: 2023-11-20

---

## Kurator's Rollout design

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

To further enhance Kurator's functionality, this proposal designs Kurator's Rollout feature to meet user's need for automatically validate released code.

By integrating Flagger, we aim to provide our users with reliable, fast and unified release validation capabilities. Enabling them to easily validate distribution code across multiple clusters.

Base on Flagger, Kurator also offers A/B Testing, Blue/Green and Canary distribution options. Meet the diverse needs of the Unified Rollout.

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

Unified rollout only requires the user to declare the required API configuration in one place, and Kurator implements subsequent validation releases based on that configuration.

In Kurator, you can choose to distribute applications with the same configuration to multiple clusters for authentication. Test new features in different cluster environments with default or custom metrics. Reduce manual effort by automatically publishing when tests succeed and rolling back when tests fail.

- **Unified Rollout**
    - Supports unified configuration of releases for multiple clusters. Achieve the deployment configuration of the application to be distributed to the specified single or multiple clusters.
    - Supports A/B, Blue/Green, and Canary releases and performs health checks based on set metrics.
    - Supports automatic rollback when release validation fails.

#### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- **Traffic distribution tools other than istio are not supported** While Flagger is able to support a wide range of traffic distribution tools including istio, nginx for grey scale releases. However, Kurator currently only supports unified management of istio across clusters. Kurator may implement other traffic distribution tools in the future.

### Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->
The purpose of this proposal is to introduce a unified Rollout for Kurator that supports A/B, Blue Green, and Canary.The main objectives of this proposal are as follows:

Application Programming Interface (API): Design API to enable Uniform Rollout. Provide an API interface for defining configuration distribution rules for unified configuration distribution by extending the fields of application.

Rollout Manager: The Rollout Manager is responsible for monitoring what is going on in the Application CRDs in the cluster and performing defined functions.

By integrating these enhancements, Kurator will provide users with a powerful and streamlined solution for managing the task of implementing Unified Configuration Distribution and simplifying the overall operational process.

#### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

##### Story 1

**User Role**: Cloud Native Project Development Team.

**Feature**: With the enhanced Kurator, developers can easily deploy their new releases to multiple clusters for validation testing.

**Value**: Provides a simplified automated way to uniformly manage configuration distribution and grey scale releases across multiple clusters. Validate new features of the product across multiple clusters in different environments. Avoid duplicate configurations and reduce workload.

**Outcome**: With this feature, developers can easily assign uniform configurations to multiple clusters to improve reliability, availability, and storage efficiency for business publishing and easily achieve scalability.

##### Story 2

**User Role**: Application Operator.

**Feature**: With the enhanced Kurator, developers can quickly release and A/B, Blue/Green or Canary test new requirements in their environment after they are completed.

**Value**: Provides a simplified, automated way for developers to distribute configurations in a uniform manner. Enables validation testing in multiple usage environments. Provides A/B, Blue/Green or Canary tests to meet different testing needs.

**Outcome**: With this feature, developers can easily assign uniform configurations to multiple clusters, test new releases, and ensure the quality of new releases. In addition, it also provides automatic rollback function when the test fails, reducing the developer's operational burden and bug impact time.

### Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

In this section, we'll dive into the detailed API design for the Unified Rollout Feature.

These APIs are designed to facilitate Kurator's integration with Flagger to enable the required functionality.

Unlike Flagger, we may need to adjust Unified Rollout to reflect our new strategy and decisions.

#### Unified Rollout API

Kurator is designed to unify the installation of Flagger as a fleet plugin in a given single or multiple clusters.

Then use the Kurator application to distribute the Flagger configuration. Kurator's unified configuration distribution.

Kurator puts the Rollout's api under the [Application](https://github.com/kurator-dev/kurator/blob/main/pkg/apis/apps/v1alpha1/types.go) CRD, so that when Kurator deploys the workload in the target cluster, it also deploys the corresponding Rollout policy.

Here's the preliminary design for the Unified Rollout:

```console
// ApplicationSyncPolicy defines the configuration to sync an artifact.
// Only oneof `kustomization` or `helm` can be specified to manage application sync.
// ApplicationSyncPolicy distributes the Rollout configuration 
// at the same time as the application deployment, if needed. 
type ApplicationSyncPolicy struct {
    // Rollout defines the rollout Configurations to be used.
    // If specified, a uniform Rollout policy is configured for this installed object.
    // +optional
    Rollout *RolloutConfig `json:"rolloutPolicy,omitempty"`
}

type RolloutConfig struct {
    // Testloader defines Whether to install testloader for users. Default is true.
    // Testloader generates traffic during canary analysis.
    // If set it to false, user need to install the testloader himself.
    // If set it to true or leave it blank, Kurator will install the flagger's testloader.
    // +optional
    TestLoader bool `json:"testLoader,omitempty"`

    // Kurator only supports istio for now.
    // New Provider will be added later.
    // +optional
    TrafficRoutingProvider string `json:"trafficRoutingProvider,omitempty"`

    // Workload specifies what workload to deploy the test to. 
    // Workload of type deployment or daemonSet.
    Workload *WorkloadReference `json:"workload"`

    // ServiceName holds the name of a service which matches the `Workload`.
    ServiceName string `json:"serviceName"`

    // Port of the workload's Service which traffic access.
    Port int32 `json:"port"`

    // Primary is the labels and annotations to add to the primary service.
    // Primary service is stable service.
    // +optional
    Primary *CustomMetadata `json:"primary,omitempty"`

    // Canary is the labels and annotations to add to the canary service.
    // Canary service is preview service.
    // +optional
    Preview *CustomMetadata `json:"preview,omitempty"`

    // RolloutPolicy defines the Release Strategy of workload.
    RolloutPolicy *RolloutPolicy `json:"rolloutPolicy"`
}
```

`Testloader` indicates whether the user wants to install the test traffic load themselves. If you don't want to install the Testloader yourself, Kurator will install flagger's Testloader by default.

`RolloutPolicy` defines the Rollout configuration for this installation workload. Although there is no detailed distinction in Kurator between canary, A/B testing and blue-green, giving users the freedom to configure traffic rules. Complete the release test. However, it is not allowed to configure canary and A/B or blue-green for the same workload.

```console
// Note: refer to https://github.com/fluxcd/flagger/blob/main/pkg/apis/flagger/v1beta1/canary.go
type RolloutPolicy struct {
    // Checknum defines the number of checks to run for A/B Testing and Blue/Green
    // Note: Kurator determines whether blue-green or A/B related processing is required based on 
    // the presence or absence of content in the Checknum field. 
    // So can't configure Iterations and CanaryStrategy at the same time.
    // +optional
    CheckNum int `json:"checknum,omitempty"`

    // The TrafficRouting defines the configuration of the gateway, traffic distribution rules, and so on.
    TrafficRouting *TrafficRoutingConfig `json:"trafficRouting"`

    // TrafficAnalysis defines the validation process of a release
    TrafficAnalysis *TrafficAnalysis `json:"trafficAnalysis,omitempty"`

    // ProgressDeadlineSeconds represents the maximum time in seconds for a
    // canary deployment to make progress before it is considered to be failed.
    // Defaults to 600s.
    // +optional
    ProgressDeadlineSeconds *int32 `json:"progressDeadlineSeconds,omitempty"`

    // SkipAnalysis promotes the canary without analyzing it
    // +optional
    SkipTrafficAnalysis bool `json:"skipTrafficAnalysis,omitempty"`

    // Restore resources to initial state when deleting canary resources.
    // Use of the revertOnDeletion property should be enabled
    // when you no longer plan to rely on Kurator for deployment management.
    // +optional
    RevertOnDeletion bool `json:"revertOnDeletion,omitempty"`

    // Suspend, if set to true will suspend the Canary, disabling any canary runs
    // regardless of any changes to its target, services, etc. Note that if the
    // Canary is suspended during an analysis, its paused until the Canary is uninterrupted.
    // +optional
    Suspend bool `json:"suspend,omitempty"`
}
```

WorkloadReference contains enough information to let you locate the typed referenced object in the same namespace. The two types of Kind now supported are `Deployment` and `DaemonSet`.

```console
type WorkloadReference struct {
    // API version of the referent
    // +optional
    APIVersion string `json:"apiVersion,omitempty"`

    // Kind of the referent.
    // Support Deployment and DaemonSet.
    Kind string `json:"kind"`

    // Namespace of the referent
    Namespace string `json:"namespace"`

    // Name of the referent
    Name string `json:"name"`
}
```

Kurator will create a VirtualService resource based on the configuration in `VirtualServiceConfig` to distribute traffic.

```console
// Note: refer to https://github.com/fluxcd/flagger/blob/main/pkg/apis/flagger/v1beta1/canary.go
type TrafficRoutingConfig struct {
    // Timeout of the HTTP or gRPC request.
    // Timeout in upstream response time.
    // +optional
    Timeout string `json:"timeout,omitempty"`

    // Gateways attached to the generated Istio virtual service.
    // Defaults to the internal mesh gateway.
    // +optional
    Gateways []string `json:"gateways,omitempty"`

    // Threshold defines the Max number of failed checks before the rollout is terminated.
    Threshold int `json:"threshold"`

    // Defaults to the RolloutConfig.ServiceName
    // +optional
    Hosts []string `json:"hosts,omitempty"`

    // Match conditions of HTTP header.
    // +optional
    Match []istiov1alpha3.HTTPMatchRequest `json:"match,omitempty"`

    // Retries policy for Http links.
    // +optional
    Retries *istiov1alpha3.HTTPRetry `json:"retries,omitempty"`

    // Headers operations for the Request.
    // e.g.
    // headers:
    //   request:
    //     add:
    //       x-some-header: "value"
    // +optional
    Headers *istiov1alpha3.Headers `json:"headers,omitempty"`

    // Cross-Origin Resource Sharing policy for the Request.
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

    // CanaryStrategy defines parameters for canary test.
    CanaryStrategy CanaryConfig `json:"canaryStrategy,omitempty"`
}

type CanaryConfig struct {
    // Max traffic weight routed to canary test
    // +optional
    MaxWeight int `json:"maxWeight,omitempty"`

    // StepWeight defines the incremental traffic weight step for analysis phase
    // If set stepWeight: 10 and set maxWeight: 50
    // The flow ratio between PREVIEW and PRIMARY at each step is
    // (10:90) (20:80) (30:70) (40:60) (50:50)
    // +optional
    StepWeight int `json:"stepWeight,omitempty"`

    // StepWeights defines the incremental traffic weight steps for analysis phase
    // Note: Cannot configure StepWeights and StepWeight at the same time.
    // If both StepWeights and MaxWeight are configured, the traffic 
    // will be scaled according to the settings in StepWeights only.
    // If set stepWeights: [1, 10, 20, 80]
    // The flow ratio between PREVIEW and PRIMARY at each step is
    // (1:99) (10:90) (20:80) (80:20)
    // +optional
    StepWeights []int `json:"stepWeights,omitempty"`

    // StepWeightPromotion defines the incremental traffic weight step for promotion phase
    // If maxWeight: 50 and set StepWeightPromotion: 20
    // After a successful test, traffic to the PRIMARY version changes as follows: 50 70 90 100.
    // +optional
    StepWeightPromotion int `json:"stepWeightPromotion,omitempty"`
}
```

As part of the TrafficAnalysis process, Kurator can validate service level objectives (SLOs) like availability, error rate percentage, average response time and any other objective based on app specific metrics. If a drop in performance is noticed during the SLOs analysis, the release will be automatically rolled back with minimum impact to end-users.

```console
// Note: refer to https://github.com/fluxcd/flagger/blob/main/pkg/apis/flagger/v1beta1/canary.go
type TrafficAnalysis struct {
    // Schedule interval for this traffic analysis
    Interval string `json:"interval"`

    // Max number of failed checks before the traffic analysis is terminated
    Threshold int `json:"threshold"`

    // Metric check list for this traffic analysis
    // Flagger comes with two builtin metric checks: HTTP request success rate and duration.
    // Can use either built-in metric checks or custom checks.
    // If you want use custom checks, you can refer to https://docs.flagger.app/usage/metrics#custom-metrics.
    // +optional
    Metrics []Metric `json:"metrics,omitempty"`

    // Webhook list for this traffic  analysis
    // +optional
    Webhooks []Webhook `json:"webhooks,omitempty"`

    // SessionAffinity represents the session affinity settings for a analysis run.
    // +optional
    SessionAffinity *SessionAffinity `json:"sessionAffinity,omitempty"`
}

type Metric struct {
    // Name of the metric.
    // User also can use Name point to custom metric checks.
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

// CustomMetadata holds labels and annotations to set on generated objects.
type CustomMetadata struct {
    Labels      map[string]string `json:"labels,omitempty"`
    Annotations map[string]string `json:"annotations,omitempty"`
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

End-to-End Tests: Comprehensive E2E tests should be conducted to ensure the  Rollout processes work seamlessly across different clusters.

Integration Tests: Integration tests should be designed to ensure Kurator's integration with Flagger functions as expected.

Unit Tests: Unit tests should cover the core functionalities and edge cases.

Isolation Testing: The Rollout processes functionalities should be tested in isolation and in conjunction with other components to ensure compatibility and performance.

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

Consideration: Integrating with other existing Rollout tools like Argo CD, Argo Rollout was also considered.

Rationale for Rejection: While these tools are powerful, they may not offer the same level of customization and Kubernetes-native capabilities as Flagger.
Additionally, this approach would have required extensive modifications to align with the cloud-native focus of the Kurator project.
