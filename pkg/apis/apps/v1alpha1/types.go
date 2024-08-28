/*
Copyright Kurator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	flaggerv1b1 "github.com/fluxcd/flagger/pkg/apis/flagger/v1beta1"
	istiov1alpha3 "github.com/fluxcd/flagger/pkg/apis/istio/v1alpha3"
	helmv2beta1 "github.com/fluxcd/helm-controller/api/v2beta1"
	kustomizev1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,categories=kurator-dev
// +kubebuilder:subresource:status

// Application is the schema for the application's API.
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ApplicationSpec   `json:"spec,omitempty"`
	Status            ApplicationStatus `json:"status,omitempty"`
}

// ApplicationSpec defines the configuration to produce an artifact and how to dispatch it.
type ApplicationSpec struct {
	// Source defines the artifact source.
	Source ApplicationSource `json:"source"`
	// SyncPolicies controls how the artifact will be customized and where it will be synced.
	SyncPolicies []*ApplicationSyncPolicy `json:"syncPolicies"`
	// Destination defines the destination clusters where the artifacts will be synced.
	// It can be overridden by the syncPolicies' destination.
	// And if both the current field and syncPolicies' destination are empty, the application will be deployed directly in the cluster where kurator resides.
	// +optional
	Destination *ApplicationDestination `json:"destination,omitempty"`
}

// ApplicationSource defines the configuration to produce an artifact for git, helm or oci repository.
// Note only one source can be specified.
type ApplicationSource struct {
	// +optional
	GitRepository *sourcev1beta2.GitRepositorySpec `json:"gitRepository,omitempty"`
	// +optional
	HelmRepository *sourcev1beta2.HelmRepositorySpec `json:"helmRepository,omitempty"`
	// +optional
	OCIRepository *sourcev1beta2.OCIRepositorySpec `json:"ociRepository,omitempty"`
}

// ApplicationDestination defines the configuration to dispatch an artifact to a fleet or specific clusters.
type ApplicationDestination struct {
	// Fleet defines the fleet to dispatch the artifact.
	// +required
	Fleet string `json:"fleet"`
	// ClusterSelector specifies the selectors to select the clusters within the fleet.
	// If unspecified, all clusters in the fleet will be selected.
	// +optional
	ClusterSelector *ClusterSelector `json:"clusterSelector,omitempty"`
}

type ClusterSelector struct {
	// MatchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value".
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	// +optional
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// ApplicationSyncPolicy defines the configuration to sync an artifact.
// Only oneof `kustomization` or `helm` can be specified to manage application sync.
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

	// Rollout defines the rollout configurations to be used.
	// If specified, a uniform rollout policy is configured for this installed object.
	// +optional
	Rollout *RolloutConfig `json:"rollout,omitempty"`
}

type RolloutConfig struct {
	// Testloader defines whether to install a private testloader for Kurator.
	// Testloader generates traffic during rollout analysis.
	// Default is false. Because Kurator will installs a public testloader with the flagger installation.
	// If set it to true, Kurator will install a private testloader dedicated to requesting the workload.
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

type RolloutPolicy struct {
	// TrafficRouting defines the configuration of the gateway, traffic routing rules, and so on.
	TrafficRouting *TrafficRoutingConfig `json:"trafficRouting,omitempty"`

	// TrafficAnalysis defines the validation process of a release
	TrafficAnalysis *TrafficAnalysis `json:"trafficAnalysis,omitempty"`

	// RolloutTimeoutSeconds represents the maximum time in seconds for a
	// preview deployment to make progress before it is considered to be failed.
	// Defaults to 600.
	// +optional
	RolloutTimeoutSeconds *int32 `json:"rolloutTimeoutSeconds,omitempty"`

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
	//
	// ```yaml
	// retries:
	//   attempts: 3
	//   perTryTimeout: 2s
	//   retryOn: gateway-error,connect-failure,refused-stream
	// ```
	//
	// +optional
	Retries *istiov1alpha3.HTTPRetry `json:"retries,omitempty"`

	// Headers operations for the Request.
	// e.g.
	//
	// ```yaml
	// headers:
	//   request:
	//     add:
	//       x-some-header: "value"
	// ```
	//
	// +optional
	Headers *istiov1alpha3.Headers `json:"headers,omitempty"`

	// Cross-Origin Resource Sharing policy for the request.
	// e.g.
	//
	// ```yaml
	// corsPolicy:
	//   allowHeaders:
	//   - x-some-header
	//   allowMethods:
	//   - GET
	//   allowOrigin:
	//   - example.com
	//   maxAge: 24h
	// ```
	//
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
	// +kubebuilder:validation:Minimum=0
	// +optional
	AnalysisTimes int `json:"analysisTimes,omitempty"`

	// Match conditions of A/B Testing HTTP header.
	// The header keys must be lowercase and use hyphen as the separator.
	// values are case-sensitive and formatted as follows:
	// - `exact: "value"` for exact string match
	// - `prefix: "value"` for prefix-based match
	// - `regex: "value"` for ECMAscript style regex-based match
	// e.g.:
	//
	// ```yaml
	// match:
	//   - headers:
	//       myheader:
	//         regex: ".*XXXX.*"
	//   - headers:
	//       cookie:
	//         regex: "^(.*?;)?(type=insider)(;.*)?$"
	// ```
	//
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
	Webhooks Webhook `json:"webhooks,omitempty"`

	// SessionAffinity represents the session affinity settings for a analysis run.
	// +optional
	SessionAffinity *SessionAffinity `json:"sessionAffinity,omitempty"`
}

type Metric struct {
	// Name of the metric.
	// Currently internally supported metric are `request-success-rate` and `request-duration`.
	// And you can use the metrics that come with the gateway.
	// When you define a metric rule in `CustomMetric`, fill in the custom name in this field.
	Name MetricName `json:"name"`

	// IntervalSeconds defines metrics query interval.
	// Defaults to 60.
	IntervalSeconds *int `json:"intervalSeconds,omitempty"`

	// ThresholdRange defines valid value accepted for this metric.
	// If no thresholdRange are set, Kurator will default every check is successful.
	// +optional
	ThresholdRange *CanaryThresholdRange `json:"thresholdRange,omitempty"`

	// CustomMetric defines the metric template to be used for this metric.
	// +optional
	CustomMetric *flaggerv1b1.MetricTemplateSpec `json:"customMetric,omitempty"`
}

type MetricName string

const (
	// `request-success-rate` indicates the successful request ratio during this checking intervalSeconds.
	// It returns a value from 0 to 100.
	RequestSuccessRate MetricName = "request-success-rate"
	// `request-duration` indicates P99 latency of the requests during the check interval.
	// `request-duration` returns in milliseconds.
	RequestDuration MetricName = "request-duration"
)

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
//
// ```yaml
// webhooks:
//   - timeoutSeconds: 15
//     command:
//   - "hey -z 1m -q 10 -c 2 http://podinfo-canary.test:9898/"
//
// ```
//
// The above example means that during trafficAnalysis, the cmd of "http://flagger-loadtester.test/" is invoked
// to execute the command "hey -z 1m -q 10 -c 2 http://podinfo-canary.test:9898/"
type Webhook struct {
	// TimeoutSeconds defines request timeout for this webhook
	// Defaults to 60
	TimeoutSeconds *int `json:"timeoutSeconds,omitempty"`

	// Commands define to commands that executed by webhook.
	// +optional
	Commands []string `json:"command,omitempty"`
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

// ApplicationStatus defines the observed state of Application.
type ApplicationStatus struct {
	SourceStatus *ApplicationSourceStatus `json:"sourceStatus,omitempty"`
	SyncStatus   []*ApplicationSyncStatus `json:"syncStatus,omitempty"`
}

// applicationSourceStatus defines the observed state of the artifact source.
type ApplicationSourceStatus struct {
	GitRepoStatus  *sourcev1beta2.GitRepositoryStatus  `json:"gitRepoStatus,omitempty"`
	HelmRepoStatus *sourcev1beta2.HelmRepositoryStatus `json:"helmRepoStatus,omitempty"`
	OCIRepoStatus  *sourcev1beta2.OCIRepositoryStatus  `json:"ociRepoStatus,omitempty"`
}

// ApplicationSyncStatus defines the observed state of Application sync.
type ApplicationSyncStatus struct {
	Name                string                                `json:"name,omitempty"`
	KustomizationStatus *kustomizev1beta2.KustomizationStatus `json:"kustomizationStatus,omitempty"`
	HelmReleaseStatus   *helmv2beta1.HelmReleaseStatus        `json:"HelmReleaseStatus,omitempty"`
	RolloutStatus       *RolloutStatus                        `json:"rolloutStatus,omitempty"`
}

// RolloutStatus defines the observed state of Rollout.
type RolloutStatus struct {
	// ClusterName is the Name of the cluster where the rollout is being performed.
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// RolloutNameInCluster is the name of the rollout being performed within this cluster.
	// +optional
	RolloutNameInCluster string `json:"rolloutNameInCluster,omitempty"`

	// RolloutStatusInCluster is the current status of the Rollout performed within this cluster.
	// +optional
	RolloutStatusInCluster *flaggerv1b1.CanaryStatus `json:"rolloutStatusInCluster,omitempty"`
}

// ApplicationList contains a list of Application.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}
