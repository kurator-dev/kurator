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
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FleetPhase string

const (
	// PendingPhase is the phase when the fleet is not processed.
	PendingPhase FleetPhase = "Pending"
	// RunningPhase is the phase when fleet control plane is being installed.
	RunningPhase FleetPhase = "Running"
	// FailedPhase is the phase when fleet control plane installation installed.
	FailedPhase FleetPhase = "Failed"
	// ReadyPhase is the phase when fleet control plane installation finished successfully.
	ReadyPhase FleetPhase = "Ready"
	// TerminatingPhase is the phase when fleet control plane is terminating.
	TerminatingPhase FleetPhase = "Terminating"
	// TerminateFailedPhase is the phase when fleet control plane terminate failed.
	TerminateFailedPhase FleetPhase = "TerminateFailed"
	// TerminateSucceededPhase is the phase when fleet control plane is terminated successfully.
	TerminateSucceededPhase FleetPhase = "TerminateSucceeded"
)

// ControlplaneAnnotation is the annotation that can be added to the fleet
// to indicate fleet manager to install control plane for the fleet.
// Current the supported value of the annotation is `karmada`.
const ControlplaneAnnotation = "fleet.kurator.dev/controlplane"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,categories=kurator-dev
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase of the fleet"

// Fleet represents a group of clusters, it is to consistently manage a group of clusters.
type Fleet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FleetSpec   `json:"spec,omitempty"`
	Status            FleetStatus `json:"status,omitempty"`
}

// FleetSpec defines the desired state of the fleet
type FleetSpec struct {
	// Clusters represents the clusters that would be registered to the fleet.
	// +required
	Clusters []*corev1.ObjectReference `json:"clusters,omitempty"`

	// Plugin defines the plugins that would be installed in the fleet.
	// +optional
	Plugin *PluginConfig `json:"plugin,omitempty"`
}

type PluginConfig struct {
	// Metric defines the configuration for the monitoring system installation and metrics collection..
	// +optional
	Metric *MetricConfig `json:"metric,omitempty"`
	// Grafana defines the configuration for the grafana installation and observation.
	// +optional
	Grafana *GrafanaConfig `json:"grafana,omitempty"`
	// Policy defines the configuration for the ploicy management.
	Policy *PolicyConfig `json:"policy,omitempty"`
	// Backup defines the configuration for the backup engine(Velero).
	Backup *BackupConfig `json:"backup,omitempty"`
}

type MetricConfig struct {
	// Thanos defines the configuration for the thanos querier and store gateway.
	Thanos ThanosConfig `json:"thanos,omitempty"`
	// Prometheus defines the configuration for the prometheus installation
	// in the clusters observed by the thanos,
	// by default thanos sidecar will be installed in thanos sidecar mode.
	Prometheus PrometheusConfig `json:"prometheus,omitempty"`
}

type PrometheusConfig struct {
	// Chart defines the helm chart config of the prometheus.
	// default values is
	//
	// chart:
	//   repository: oci://registry-1.docker.io/bitnamicharts
	//   name: kube-prometheus
	//   version: 8.9.1
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`
	// NodeExporter defines the configuration for the node exporter.
	// +optional
	NodeExporter *PrometheusExporterConfig `json:"nodeExporter,omitempty"`
	// KubeStateMetrics defines the configuration for the kube-state-metrics.
	// +optional
	KubeStateMetrics *PrometheusExporterConfig `json:"kubeStateMetrics,omitempty"`
	// ExtraArgs is the set of extra arguments for Prometheus chart.
	//
	// For Example, using following configuration to create a ServiceMonitor to monitor prometheus itself.
	// extraArgs:
	//   prometheus:
	//     serviceMonitor:
	//       enabled: true
	//
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}

type PrometheusExporterConfig struct {
	// Enabled indicates whether the exporters are enabled.
	Enabled bool `json:"enabled,omitempty"`
}

type ChartConfig struct {
	// Repository defines the repository of chart.
	// Default value depends on the kind of the component.
	// +optional
	Repository string `json:"repository,omitempty"`
	// Name defines the name of the chart.
	// Default value depends on the kind of the component.
	// +optional
	Name string `json:"name,omitempty"`
	// Version defines the version of the chart.
	// Default value depends on the kind of the component.
	// +optional
	Version string `json:"version,omitempty"`
}

type ThanosConfig struct {
	// Chart defines the helm chart config of the thanos.
	// default values is
	//
	// chart:
	//   repository: oci://registry-1.docker.io/bitnamicharts
	//   name: thanos
	//   version: 12.5.1
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`
	// ObjectStoreConfig is the secret reference of the object store.
	// Configuration must follow the definition of the thanos: https://thanos.io/tip/thanos/storage.md/.
	// +required
	ObjectStoreConfig ObjectStoreConfig `json:"objectStoreConfig"`
	// ExtraArgs is the set of extra arguments for Thanos chart.
	//
	// For Example, using following configuration to enable query frontend.
	// extraArgs:
	//   queryFrontend:
	//     enabled: true
	//
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}

type ObjectStoreConfig struct {
	// SecretName is the name of the secret that holds the object store configuration.
	// The path of object store configuration must be `objstore.yml`
	// +required
	SecretName string `json:"secretName"`
}

type GrafanaConfig struct {
	// Chart defines the helm chart config of the grafana.
	// default values is
	//
	// chart:
	//   repository: oci://registry-1.docker.io/bitnamicharts
	//   name: grafana
	//   version: 8.2.33
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`
	// ExtraArgs is the set of extra arguments for Grafana chart.
	//
	// For Example, using following configuration to change replica count.
	// extraArgs:
	//   grafana:
	//     replicaCount: 2
	//
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}

type PolicyConfig struct {
	// Kyverno defines the configuration for the kyverno installation and policy management.
	// +optional
	Kyverno *KyvernoConfig `json:"kyverno,omitempty"`

	// TODO: support other policy management system.
}

type KyvernoConfig struct {
	// Chart defines the helm chart config of the kyverno.
	// default values is
	// chart:
	//   repository: https://kyverno.github.io/kyverno/
	//   name: kyverno
	//   version: 3.0.0
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`
	// PodSecurity defines the pod security configuration for the kyverno.
	// +optional
	PodSecurity *PodSecurityPolicy `json:"podSecurity,omitempty"`
	// ExtraArgs is the set of extra arguments for Grafana chart.
	//
	// For Example, using following configuration to change image pull policy.
	// extraArgs:
	//   image:
	//     pullPolicy: Always
	//
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}

type PodSecurityPolicy struct {
	// Standard defines the pod security standard.
	// More details: https://kubernetes.io/docs/concepts/security/pod-security-standards
	// +kubebuilder:validation:Enum=privileged;baseline;restricted
	// +kubebuilder:default=baseline
	// +optional
	Standard string `json:"standard,omitempty"`
	// Severity indicates policy check result criticality in a policy report.
	// +kubebuilder:validation:Enum=low;medium;high
	// +kubebuilder:default=medium
	// +optional
	Severity string `json:"severity,omitempty"`
	// ValidationFailureAction indicates the action to take when a pod creation fails to validate.
	// For more info https://kyverno.io/docs/writing-policies/validate/#validation-failure-action
	// +kubebuilder:validation:Enum=Enforce;Audit
	// +kubebuilder:default=Audit
	// +optional
	ValidationFailureAction string `json:"validationFailureAction,omitempty"`
}

// BackupConfig defines the configuration for backups.
type BackupConfig struct {
	// Chart defines the helm chart configuration of the backup engine.
	// The default value is:
	//
	// chart:
	//   repository: https://vmware-tanzu.github.io/helm-charts
	//   name: velero
	//   version: 5.0.2
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`

	// Storage provides details on where the backup data should be stored.
	Storage BackupStorage `json:"storage"`

	// ExtraArgs provides the extra chart values for the backup engine chart.
	// For example, use the following configuration to change the image tag or pull policy:
	//
	// extraArgs:
	//   image:
	//     repository: velero/velero
	//     tag: v1.11.1
	//     pullPolicy: IfNotPresent
	//
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}

type BackupStorage struct {
	// Location specifies where the backup data will be stored.
	Location BackupStorageLocation `json:"location"`

	// Credentials refers to the Kubernetes secret containing the AccessKeyID and SecretAccessKey
	// required to access the backup storage location. The secret might, for example,
	// contain fields such as `accessKeyID` and `secretAccessKey` to store the credentials.
	Credentials string `json:"credentials"`
}


type BackupStorageLocation struct {
	// Bucket specifies the storage bucket name.
	Bucket string `json:"bucket"`
	// Provider specifies the storage provider type (e.g., aws).
	Provider string `json:"provider"`
	// S3Url provides the endpoint URL for S3-compatible storage.
	S3Url string `json:"s3Url"`
	// Region specifies the region of the storage.
	Region string `json:"region"`
}

// FleetStatus defines the observed state of the fleet
type FleetStatus struct {
	// CredentialSecret is the secret name that holds credentials used for accessing the fleet control plane.
	CredentialSecret *string `json:"credentialSecret,omitempty"`

	// Phase represents the current phase of fleet.
	// E.g. Pending, Running, Terminating, Failed, Ready, etc.
	// +optional
	Phase FleetPhase `json:"phase,omitempty"`

	// TODO: add conditions fields if needed

	// A brief CamelCase message indicating details about why the fleet is in this state.
	// +optional
	Reason string `json:"reason,omitempty"`

	// PluginEndpoints is the endpoints of the plugins.
	PluginEndpoints map[string]Endpoints `json:"pluginEndpoints,omitempty"`

	// Total number of ready clusters, ready to deploy .
	ReadyClusters int32 `json:"readyClusters,omitempty"`

	// Total number of unready clusters, not ready for use.
	UnReadyClusters int32 `json:"unReadyClusters,omitempty"`
}

type Endpoints []string

// FleetList contains a list of fleets.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FleetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Fleet `json:"items"`
}
