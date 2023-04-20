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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,categories=kurator-dev
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase of the Telemetry"

// Telemetry represents the configuration of telemetry within a fleet.
type Telemetry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TelemetrySpec   `json:"spec,omitempty"`
	Status            TelemetryStatus `json:"status,omitempty"`
}

// TelemetrySpec defines the desired state of Telemetry.
type TelemetrySpec struct {
	// FleetName is the name of the fleet that this Telemetry belongs to.
	// This field is immutable.
	FleetName string `json:"fleetName,omitempty"`
	// Metric defines the configuration for the monitoring system installation and metrics collection..
	// +optional
	Metric *MetricConfig `json:"metric,omitempty"`
	// Grafana defines the configuration for the grafana installation and observation.
	// +optional
	Grafana *GrafanaConfig `json:"grafana,omitempty"`

	// TODO: add logging/tracing config?
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
	// OCI registry is not supported.
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
	// Mode defines the mode of the Thanos.
	// The default is sidecar.
	// +kubebuilder:validation:Enum=sidecar;reciever
	// +optional
	Mode string `json:"mode,omitempty"`
	// ObjectStoreConfig defines the configuration for the object store.
	ObjectStore *ThanosObjectStoreConfig `json:"objectStore,omitempty"`
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

type ThanosObjectStoreConfig struct {
	// Type defines the type of the object store.
	// Now only support s3, which means S3-compatible object storages, e.g. MinIO.
	Type string `json:"type"`
	// S3 defines the configuration for the S3 object store.
	// For more information, see https://thanos.io/tip/thanos/storage.md/#s3
	// This field is required if the type is s3.
	// +optional
	S3 *S3Config `json:"s3,omitempty"`
}

// FilesystemConfig defines the configuration for storing and accessing blobs in filesystem.
type FilesystemConfig struct {
	Directory string `json:"directory"`
}

type S3Config struct {
	// Bucket defines the name of the bucket.
	Bucket string `json:"bucket"`
	// Region defines the region of the bucket.
	Region string `json:"region"`
	// Endpoint defines the endpoint of the bucket.
	Endpoint string `json:"endpoint"`
	// Credential is the credential used to access the bukcet.
	// Make sure the credential is in the same namespace as the monitoring.
	// The secret must have following keys: AccessKeyID, SecretAccessKey.
	// SessionToken is optional.
	Credential clusterv1alpha1.CredentialConfig `json:"credential"`
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

// TelemetryStatus defines the observed state of Telemetry.
type TelemetryStatus struct {
	// Phase represents the current phase of Telemetry.
	// E.g. Pending, Running, Terminating, Failed, Ready, etc.
	// +optional
	Phase string `json:"phase,omitempty"`
	// A brief message indicating details about why the Telemetry is in this state.
	// +optional
	Reason string `json:"reason,omitempty"`
}

type Endpoints map[string]string

// TelemetryList contains a list of telemetries.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TelemetryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Telemetry `json:"items"`
}
