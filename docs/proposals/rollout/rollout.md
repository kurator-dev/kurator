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

Base on Flagger, Kurator also offers A/B Testing, Blue/Green Deployment and Canary Deployment distribution options. Meet the diverse needs of the Unified Rollout.

### Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users.
-->

With the increase in project size and complexity and the development of cloud computing technology, the CI/CD process has been proposed.

CI/CD has many advantages such as increased development efficiency, improved quality, more reliable deployment and better continuous learning and improvement, which is more suitable for today's software development process.

Therefore, CI/CD as an important feature of cloud native usage scenarios, Kurator needs to provide relevant functional support to achieve the vision of Kurator unified configuration distribution.

#### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

Unified rollout only requires the user to declare the required API configuration in one place, and Kurator implements subsequent validation releases based on that configuration.

In Kurator, you can choose to distribute applications with the same configuration to multiple clusters for validating new features. Test new features in different cluster environments with default or custom metrics. Reduce manual effort by automatically publishing when tests succeed and rolling back when tests fail.

- **Unified Rollout**
    - Supports unified configuration of releases for multiple clusters. Achieve the deployment configuration of the application to be distributed to the specified single or multiple clusters.
    - Supports A/B Testing, Blue/Green Deployment, and Canary Deployment and performs health checks based on setting metrics.
    - Supports automatic rollback when release validation fails.

#### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- **Traffic distribution tools other than istio are not supported** While Flagger is able to support a wide range of traffic routing tools including istio, nginx and so on for rollout testing. However, Kurator currently only take Istio to be traffic routing provider. Kurator may implement other traffic routing tools in the future.

### Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->
The purpose of this proposal is to introduce a unified Rollout for Kurator that supports A/B Testing, Blue/Green Deployment, and Canary Deployment. The main objectives of this proposal are as follows:

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

**Value**: Provides a simplified automated way to uniformly manage configuration distribution and grey releases across multiple clusters. Validate new features of the product across multiple clusters in different environments. Avoid duplicate configurations and reduce workload.

**Outcome**: With this feature, developers can easily distribute uniform configuration to multiple clusters to validate new features. Improve reliability and availability of releases and simplify work.

##### Story 2

**User Role**: Application Operators.

**Feature**: With the enhanced Kurator, developers can quickly release and A/B Testing, Blue/Green Deployment or Canary Deployment new requirements in their environment after they are completed.

**Value**: Provides a simplified, automated way for developers to distribute configurations in a uniform manner. Enables validation testing in multiple usage environments. Provides A/B Testing, Blue/Green Deployment or Canary Deployment to meet different testing needs.

**Outcome**: With this feature, developers can easily distribute uniform configuration to multiple clusters, test new releases. And ensure the quality of new releases. In addition, it also provides automatic rollback function when the test fails, reducing the developer's operational burden and bug impact time.

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
// ApplicationSyncPolicy distributes the rollout configuration 
// at the same time as the application deployment, if needed. 
type ApplicationSyncPolicy struct {
    // Rollout defines the rollout configurations to be used.
    // If specified, a uniform rollout policy is configured for this installed object.
    // +optional
    Rollout *RolloutConfig `json:"rollout,omitempty"`
}

type RolloutConfig struct {
    // Testloader defines whether to install testloader for Kurator. Default is true.
    // Testloader generates traffic during rollout analysis.
    // If set it to false, user need to install the testloader himself.
    // If set it to true or leave it blank, Kurator will install the flagger's testloader.
    // +optional
    TestLoader *bool `json:"testLoader,omitempty"`

    // TrafficRoutingProvider defines traffic routing provider.
    // Kurator only supports istio for now.
    // Other provider will be added later.
    // +optional
    TrafficRoutingProvider string `json:"trafficRoutingProvider,omitempty"`

    // Workload specifies what workload to deploy the test to. 
    // Workload of type Deployment or DaemonSet.
    Workload *CrossNamespaceObjectReference `json:"workload"`

    // ServiceName holds the name of a service which matches the workload.
    ServiceName string `json:"serviceName"`

    // Port exposed by the service.
    Port int32 `json:"port"`

    // Primary is the labels and annotations are added to the primary service.
    // Primary service is stable service. The name of the primary service in the cluster is <service name>-primary
    // +optional
    Primary *CustomMetadata `json:"primary,omitempty"`

    // Preview is the labels and annotations are added to the preview service.
    // The name of the preview service in the cluster is <service name>-canary
    // +optional
    Preview *CustomMetadata `json:"preview,omitempty"`

    // RolloutPolicy defines the release strategy of workload.
    RolloutPolicy *RolloutPolicy `json:"rolloutPolicy"`
}
```

`RolloutPolicy` defines the Rollout configuration for this Application. Although there is no detailed distinction in Kurator between Canary Deployment, A/B Testing and Blue/Green Deployment, giving users the freedom to configure traffic rules. Complete the release test. However, it is not allowed to configure Canary Deployment and A/B Testing or Blue/Green Deployment for the same workload.

```console
// Note: refer to https://github.com/fluxcd/flagger/blob/main/pkg/apis/flagger/v1beta1/canary.go
type RolloutPolicy struct {
    // TrafficRouting defines the configuration of the gateway, traffic routing rules, and so on.
    TrafficRouting *TrafficRoutingConfig `json:"trafficRouting,omitempty"`

    // TrafficAnalysis defines the validation process of a release
    TrafficAnalysis *TrafficAnalysis `json:"trafficAnalysis,omitempty"`

    // RolloutTimeoutSeconds represents the maximum time in seconds for a
    // preview deployment to make progress before it is considered to be failed.
    // Defaults to 600.
    // +optional
    RolloutTimeoutSeconds *int `json:"rolloutTimeoutSeconds,omitempty"`

    // SkipTrafficAnalysis promotes the preview release without analyzing it.
    // +optional
    SkipTrafficAnalysis bool `json:"skipTrafficAnalysis,omitempty"`

    // RevertOnDeletion defines whether to revert a resource to its initial state when deleting rollout resource.
    // Use of the revertOnDeletion property should be enabled
    // when you no longer plan to rely on Kurator for deployment management.
    // Kurator will install the Flagger to the specified cluster via a fleet plugin. 
    // If RevertOnDeletion is set to true, the Flagger will revert a resource to its initial state 
    // when the deleting Application.Spec.ApplicationSyncPolicy.Rollout or 
    // the Application.
    // +optional
    RevertOnDeletion bool `json:"revertOnDeletion,omitempty"`

    // Suspend, if set to true will suspend the rollout, disabling any rollout runs
    // regardless of any changes to its target, services, etc. Note that if the
    // rollout is suspended during an analysis, its paused until the rollout is uninterrupted.
    // +optional
    Suspend bool `json:"suspend,omitempty"`
}
```

Kurator will create a VirtualService resource based on the configuration in `VirtualServiceConfig` to distribute traffic.

```console
// Note: refer to https://github.com/fluxcd/flagger/blob/main/pkg/apis/flagger/v1beta1/canary.go
type TrafficRoutingConfig struct {
    // TimeoutSeconds of the HTTP or gRPC request.
    // +optional
    TimeoutSeconds int `json:"timeoutSeconds,omitempty"`

    // Gateways attached to the generated Istio virtual service.
    // Defaults to the internal mesh gateway.
    // +optional
    Gateways []string `json:"gateways,omitempty"`

    // Defaults to the RolloutConfig.ServiceName
    // +optional
    Hosts []string `json:"hosts,omitempty"`

    // Retries describes the retry policy to use when a HTTP request fails. 
    // For example, the following rule sets the maximum number of retries to three, 
    // with a 2s timeout per retry attempt.
    // e.g.:
    // retries:
    //   attempts: 3
    //   perTryTimeout: 2s
    //   retryOn: gateway-error,connect-failure,refused-stream
    Retries *istiov1alpha3.HTTPRetry `json:"retries,omitempty"`

    // Headers operations for the Request.
    // e.g.
    // headers:
    //   request:
    //     add:
    //       x-some-header: "value"
    // +optional
    Headers *istiov1alpha3.Headers `json:"headers,omitempty"`

    // Cross-Origin Resource Sharing policy for the request.
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

    // CanaryStrategy defines parameters for Canary Deployment.
    // Note: Kurator determines A/B Testing, Blue/Green Deployment, or Canary Deployment 
    // based on the presence of content in the canaryStrategy field.
    // So can't configure canaryStrategy and analysisTimes at the same time.
    // +optional
    CanaryStrategy *CanaryConfig `json:"canaryStrategy,omitempty"`

    // AnalysisTimes defines the number of traffic analysis checks to run for A/B Testing and Blue/Green Deployment
    // If set "analysisTimes: 10". It means Kurator will checks the preview service ten times. 
    // +optional
    AnalysisTimes int `json:"analysisTimes,omitempty"`

    // Match conditions of A/B Testing HTTP header.
    // The header keys must be lowercase and use hyphen as the separator.
    // values are case-sensitive and formatted as follows:
    // - `exact: "value"` for exact string match
    // - `prefix: "value"` for prefix-based match
    // - `regex: "value"` for ECMAscript style regex-based match
    // e.g.: 
    // match:
    //   - headers:
    //       myheader:
    //         regex: ".*XXXX.*"
    //   - headers:
    //       cookie:
    //         regex: "^(.*?;)?(type=insider)(;.*)?$"
    // Note: If you want to use A/B Testing, you need to configure analysisTimes and match. 
    // If you only configure analysisTimes, it will trigger Blue/Green Deployment.
    // You can configure both canaryStrategy and match.
    // If configure both canaryStrategy and match, Traffic that meets match goes towards the preview service. 
    // Traffic that doesn't meet the match will go to the primary service and preview service proportionally.
    // +optional
    Match []istiov1alpha3.HTTPMatchRequest `json:"match,omitempty"`
}

type CanaryConfig struct {
    // Max traffic weight routed to preview service.
    // If empty and no stepweights are set, 100 will be used by default.
    // +optional
    MaxWeight int `json:"maxWeight,omitempty"`

    // StepWeight defines the incremental traffic weight step for analysis phase
    // If set stepWeight: 10 and set maxWeight: 50
    // The flow ratio between PREVIEW and PRIMARY at each step is
    // (10:90) (20:80) (30:70) (40:60) (50:50)
    // +optional
    StepWeight int `json:"stepWeight,omitempty"`

    // StepWeights defines the incremental traffic weight steps for analysis phase
    // Note: Cannot configure stepWeights and stepWeight at the same time.
    // If both stepWeights and maxWeight are configured, the traffic 
    // will be scaled according to the settings in stepWeights only.
    // If set stepWeights: [1, 10, 20, 80]
    // The flow ratio between PREVIEW and PRIMARY at each step is
    // (1:99) (10:90) (20:80) (80:20)
    // +optional
    StepWeights []int `json:"stepWeights,omitempty"`

    // StepWeightPromotion defines the incremental traffic weight step for promotion phase
    // If maxWeight: 50 and set stepWeightPromotion: 20
    // After a successful test, traffic to the PRIMARY version changes as follows: 50 70 90 100.
    // +optional
    StepWeightPromotion int `json:"stepWeightPromotion,omitempty"`
}
```

As part of the TrafficAnalysis process, Kurator can validate service level objectives (SLOs) like availability, error rate percentage, average response time and any other objective based on app specific metrics. If a drop in performance is noticed during the SLOs analysis, the release will be automatically rolled back with minimum impact to end-users.

```console
// Note: refer to https://github.com/fluxcd/flagger/blob/main/pkg/apis/flagger/v1beta1/canary.go
type TrafficAnalysis struct {
    // CheckIntervalSeconds defines the schedule interval for this traffic analysis.
    // Interval is the time interval between each test. 
    // Kurator changes the traffic distribution rules (if they need to be changed) 
    // and performs a traffic analysis every so often.
    // Defaults to 60.
    CheckIntervalSeconds *int `json:"checkIntervalSeconds"`

    // CheckFailedTimes defines the max number of failed checks before the traffic analysis is terminated
    // If set "checkFailedTimes: 2". It means Kurator will rollback when check failed 2 times.
    // +kubebuilder:validation:Minimum=0
    CheckFailedTimes *int `json:"checkFailedTimes"`

    // Metric check list for this traffic analysis
    // Flagger comes with two builtin metric checks: HTTP request success rate and duration.
    // Can use either built-in metric checks or custom checks.
    // If you want use custom checks, you can refer to https://docs.flagger.app/usage/metrics#custom-metrics.
    // +optional
    Metrics []Metric `json:"metrics,omitempty"`

    // Webhook list for this traffic analysis
    // +optional
    Webhooks []Webhook `json:"webhooks,omitempty"`

    // SessionAffinity represents the session affinity settings for a analysis run.
    // +optional
    SessionAffinity *SessionAffinity `json:"sessionAffinity,omitempty"`
}

type Metric struct {
    // Name of the metric.
    // Currently supported metric are `request-success-rate` and `request-duration`.
    // `request-success-rate` indicates the successful request ratio during this checking intervalSeconds.
    // It returns a value from 0 to 100.
    // `request-duration` indicates P99 latency of the requests during the check interval.
    // `request-duration` returns in milliseconds.
    Name string `json:"name"`

    // IntervalSeconds defines metrics query interval.
    // Defaults to 60.
    IntervalSeconds *int `json:"intervalSeconds,omitempty"`

    // ThresholdRange defines valid value accepted for this metric.
    // +optional
    ThresholdRange *CanaryThresholdRange `json:"thresholdRange,omitempty"`
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

// Kurator generates traffic load by invoking the testloader through a webhook to request the service.
// e.g.
// webhooks:
//   - timeoutSeconds: 15
//     commend:
//       - "hey -z 1m -q 10 -c 2 http://podinfo-canary.test:9898/"
// The above example means that during trafficAnalysis, the cmd of "http://flagger-loadtester.test/" is invoked 
// to execute the command "hey -z 1m -q 10 -c 2 http://podinfo-canary.test:9898/"
type Webhook struct {
    // TimeoutSeconds defines request timeout for this webhook
    // Defaults to 60
    TimeoutSeconds *int `json:"timeoutSeconds,omitempty"`

    // Command defines to commends that executed by webhook.
    // +optional
    Command []string `json:"command,omitempty"`
}

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
