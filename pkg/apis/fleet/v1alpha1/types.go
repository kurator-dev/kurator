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
// +kubebuilder:resource:scope=Namespaced,categories=kurator-dev,path=fleets
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
	// DistributedStorage define the configuration for the distributed storage(Implemented with Rook)
	DistributedStorage *DistributedStorageConfig `json:"distributedStorage,omitempty"`
	// Flagger defines the configuretion for the kurator rollout engine.
	Flagger *FlaggerConfig `json:"flagger,omitempty"`
	// SubMarinerOperator defines the configuration for the kurator network management.
	SubMarinerOperator *SubMarinerOperatorConfig `json:"submariner,omitempty"`
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
	// default value is
	//
	// ```yaml
	// chart:
	//   repository: oci://registry-1.docker.io/bitnamicharts
	//   name: kube-prometheus
	//   version: 8.9.1
	// ```
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
	// For Example, using following configuration to create a ServiceMonitor to monitor prometheus itself.
	//
	// ```yaml
	// extraArgs:
	//   prometheus:
	//     serviceMonitor:
	//       enabled: true
	// ```
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
	// default value is
	//
	// ```yaml
	// chart:
	//   repository: oci://registry-1.docker.io/bitnamicharts
	//   name: thanos
	//   version: 12.5.1
	// ```
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`
	// ObjectStoreConfig is the secret reference of the object store.
	// Configuration must follow the definition of the thanos: https://thanos.io/tip/thanos/storage.md/.
	// +required
	ObjectStoreConfig ObjectStoreConfig `json:"objectStoreConfig"`
	// ExtraArgs is the set of extra arguments for Thanos chart.
	// For Example, using following configuration to enable query frontend.
	//
	// ```yaml
	// extraArgs:
	//   queryFrontend:
	//     enabled: true
	// ```
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
	// default value is
	//
	// ```yaml
	// chart:
	//   repository: oci://registry-1.docker.io/bitnamicharts
	//   name: grafana
	//   version: 8.2.33
	// ```
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`
	// ExtraArgs is the set of extra arguments for Grafana chart.
	// For Example, using following configuration to change replica count.
	//
	// ```yaml
	// extraArgs:
	//   grafana:
	//     replicaCount: 2
	// ```
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
	// default value is
	//
	// ```yaml
	// chart:
	//   repository: https://kyverno.github.io/kyverno/
	//   name: kyverno
	//   version: 3.0.0
	// ```
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`
	// PodSecurity defines the pod security configuration for the kyverno.
	// +optional
	PodSecurity *PodSecurityPolicy `json:"podSecurity,omitempty"`
	// ExtraArgs is the set of extra arguments for Grafana chart.
	// For Example, using following configuration to change image pull policy.
	//
	// ```yaml
	// extraArgs:
	//   image:
	//     pullPolicy: Always
	// ```
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
	// ```yaml
	// chart:
	//   repository: https://vmware-tanzu.github.io/helm-charts
	//   name: velero
	//   version: 5.0.2
	// ```
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`

	// Storage provides details on where the backup data should be stored.
	Storage BackupStorage `json:"storage"`

	// ExtraArgs provides the extra chart values for the backup engine chart.
	// For example, use the following configuration to change the image tag or pull policy:
	//
	// ```yaml
	// extraArgs:
	//   image:
	//     repository: velero/velero
	//     tag: v1.11.1
	//     pullPolicy: IfNotPresent
	// ```
	//
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}

type BackupStorage struct {
	// Location specifies where the backup data will be stored.
	Location BackupStorageLocation `json:"location"`

	// The structure of the secret varies depending on the object storage provider:
	//
	// - For AWS S3, Minio or Huawei Cloud, the secret should contain the following keys:
	//   - `access-key`: The access key for S3 authentication.
	//   - `secret-key`: The secret key for S3 authentication.
	//
	// - For GCP, the secret should be created according to the official GCP documentation.
	//   see https://github.com/vmware-tanzu/velero-plugin-for-gcp/blob/main/README.md
	//
	// - For Azure, the secret should be created according to the official Azure documentation.
	//   see https://github.com/vmware-tanzu/velero-plugin-for-microsoft-azure/blob/main/README.md
	//
	// +required
	SecretName string `json:"secretName"`
}

type BackupStorageLocation struct {
	// Bucket specifies the storage bucket name.
	Bucket string `json:"bucket"`
	// Provider specifies the storage provider type (e.g., aws, huaweicloud, gcp, azure).
	Provider string `json:"provider"`
	// Endpoint provides the endpoint URL for the storage.
	Endpoint string `json:"endpoint"`
	// Region specifies the region of the storage.
	// +optional
	Region string `json:"region,omitempty"`
	// Config is a map for additional provider-specific configurations.
	//    #  region:
	//    #  s3ForcePathStyle:
	//    #  s3Url:
	//    #  kmsKeyId:
	//    #  resourceGroup:
	//    #  The ID of the subscription containing the storage account, if different from the cluster’s subscription. (Azure only)
	//    #  subscriptionId:
	//    #  storageAccount:
	//    #  publicUrl:
	//    #  Name of the GCP service account to use for this backup storage location. Specify the
	//    #  service account here if you want to use workload identity instead of providing the key file.(GCP only)
	//    #  serviceAccount:
	//    #  Option to skip certificate validation or not if insecureSkipTLSVerify is set to be true, the client side should set the
	//    #  flag. For Velero client Command like velero backup describe, velero backup logs needs to add the flag --insecure-skip-tls-verify
	//    #  insecureSkipTLSVerify:
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

type DistributedStorageConfig struct {
	// Chart defines the helm chart configuration of the distributed storage engine.
	// The default value is:
	//
	// ```yaml
	// chart:
	//   repository: https://charts.rook.io/release
	//   name: rook
	//   version: 1.11.11
	// ```
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`

	//Storage provides detailed settings for unified distributed storage.
	Storage *DistributedStorage `json:"storage"`

	// ExtraArgs provides the extra chart values for rook chart.
	// For example, use the following configuration to change the pull policy:
	//
	// ```yaml
	// extraArgs:
	//   image:
	//     pullPolicy: Always
	// ```
	//
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}

type DistributedStorage struct {
	// The path on the host where config and data can be persisted.
	// If the storagecluster is deleted, please clean up the configuration files in this file path.
	// e.g. /var/lib/rook
	// +kubebuilder:validation:Pattern=`^/(\S+)`
	// +optional
	DataDirHostPath *string `json:"dataDirHostPath,omitempty"`

	// Monitor is the daemon that monitors the status of the ceph cluster.
	// Responsible for collecting cluster information, updating cluster information, and publishing cluster information.
	// Including monmap, osdmap, PGmap, mdsmap, etc.
	// A spec for mon related options
	// +optional
	// +nullable
	Monitor *MonSpec `json:"monitor,omitempty"`

	// Manager is the daemon runs alongside monitor daemon,to provide additional monitoring and interfaces to external monitoring and management systems.
	// A spec for mgr related options
	// +optional
	// +nullable
	Manager *MgrSpec `json:"manager,omitempty"`

	// A spec for available storage in the cluster and how it should be used
	// +optional
	// +nullable
	Storage *StorageScopeSpec `json:"storage,omitempty"`
}

type MonSpec struct {
	// Count is the number of Ceph monitors.
	// Default is three and preferably an odd number.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=9
	// +optional
	Count *int `json:"count,omitempty"`

	// The annotation-related configuration to add/set on each Pod related object. Including Pod， Deployment.
	// +nullable
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Similar to Annotation, but more graphical than Annotation.
	// The label-related configuration to add/set on each Pod related object. Including Pod， Deployment.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// The placement-related configuration to pass to kubernetes (affinity, node selector, tolerations).
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	// +optional
	Placement *Placement `json:"placement,omitempty"`
}

type MgrSpec struct {
	// Count is the number of manager to run
	// Default is two, one for use and one for standby.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=2
	// +optional
	Count *int `json:"count,omitempty"`

	// The annotation-related configuration to add/set on each Pod related object. Including Pod， Deployment.
	// +nullable
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// The label-related configuration to add/set on each Pod related object. Including Pod， Deployment.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// The placement-related configuration to pass to kubernetes (affinity, node selector, tolerations).
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	// +optional
	Placement *Placement `json:"placement,omitempty"`
}

type StorageScopeSpec struct {
	// +nullable
	// +optional
	Nodes []Node `json:"nodes,omitempty"`

	// indicating if all nodes in the cluster should be used for storage according to the cluster level storage selection and configuration values.
	// If individual nodes are specified under the nodes field, then useAllNodes must be set to false.
	// +optional
	UseAllNodes bool `json:"useAllNodes,omitempty"`

	// Select device information used by osd. For more information see the design of the selection below.
	StorageDeviceSelection `json:",inline"`
}

// Each individual node can specify configuration to override the cluster level settings and defaults.
// If a node does not specify any configuration then it will inherit the cluster level settings.
type Node struct {
	// Name should match its kubernetes.io/hostname label
	// +optional
	Name string `json:"name,omitempty"`

	// Specify which storage drives the osd deployed in this node can manage.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	// +optional
	StorageDeviceSelection `json:",inline"`
}

type StorageDeviceSelection struct {
	// List of devices to use as storage devices
	// A list of individual device names belonging to this node to include in the storage cluster
	// e.g. `sda` or  `/dev/disk/by-id/ata-XXXX`
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	// +optional
	Devices []Device `json:"devices,omitempty"`
}

// Placement is the placement for an object
type Placement struct {
	// NodeAffinity is a group of node affinity scheduling rules
	// +optional
	NodeAffinity *corev1.NodeAffinity `json:"nodeAffinity,omitempty"`
	// PodAffinity is a group of inter pod affinity scheduling rules
	// +optional
	PodAffinity *corev1.PodAffinity `json:"podAffinity,omitempty"`
	// PodAntiAffinity is a group of inter pod anti affinity scheduling rules
	// +optional
	PodAntiAffinity *corev1.PodAntiAffinity `json:"podAntiAffinity,omitempty"`
	// The pod this Toleration is attached to tolerates any taint that matches
	// the triple <key,value,effect> using the matching operator <operator>
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// TopologySpreadConstraint specifies how to spread matching pods among the given topology
	// +optional
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}

// Device represents a disk to use in the cluster
type Device struct {
	// +optional
	Name string `json:"name,omitempty"`

	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

type FlaggerConfig struct {
	// Chart defines the helm chart config of the flagger.
	// default value is
	//
	// ```yaml
	// chart:
	//   repository: oci://ghcr.io/fluxcd/charts
	//   name: flagger
	//   version: 1.x
	// ```
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`
	// ExtraArgs is the set of extra arguments for flagger chart.
	// For Example, using following configuration to change replica count.
	//
	// ```yaml
	// extraArgs:
	//   flagger:
	//     replicaCount: 2
	// ```
	//
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
	// TrafficRoutingProvider defines traffic routing provider.
	// And Kurator will install flagger in trafficRoutingProvider's namespace
	// For example, If you use `istio` as a provider, flager will be installed in istio's namespace `istio-system`.
	// Other provider will be added later.
	// +optional
	TrafficRoutingProvider Provider `json:"trafficRoutingProvider,omitempty"`
	// PublicTestloader defines whether to install the publictestloader or not.
	// In addition to the public testloader you can configure here,
	// you can also specify a private testloader in the Application.Spec.SyncPolicies.Rollout.TestLoader
	PublicTestloader bool `json:"publicTestloader,omitempty"`
}

type SubMarinerOperatorConfig struct {
	// Chart defines the helm chart configuration of the submariner operator.
	// The default value is
	//
	// ```yaml
	// chart:
	//   repository: https://submariner-io.github.io/submariner-charts/charts
	//   name: submariner-operator
	//   version: 0.18.0
	//   targetNamespace: submariner-operator
	// ```
	//
	// +optional
	Chart *ChartConfig `json:"chart,omitempty"`

	// ExtraArgs is the set of extra arguments for submariner, and example will be provided in the future.
	//
	// ```yaml
	// extraArgs:
	//   broker:
	//   		globalnet: true
	// 	 submariner:
	//  		serviceDiscovery: true
	//      natEnabled: false
	// ```
	//
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`

	// BrokerCluster is the name of cluster in which the broker will be installed.
	// If the broker cluster is not specified, the first cluster in the fleet will be used as the broker cluster.
	// +optional
	BrokerCluster string `json:"brokerCluster,omitempty"`

	// ClusterCidrs records the clustercidr of each cluster.
	ClusterCidrs map[string]string `json:"clusterCidrs"`

	// ServiceCidrs records the servicecidr of each cluster.
	ServiceCidrs map[string]string `json:"serviceCidrs"`

	// Globalcidrs records the globalcidr of each cluster in a virtual network Globalnet.
	// Each cluster must use distinct globalCidr that don’t conflict or overlap with any other cluster
	// If the globalcidr is not specified, Globalnet will be disabled.
	// +optional
	Globalcidrs map[string]string `json:"globalcidrs,omitempty"`
}

// Provider only can be istio now.
// TODO: add Linkerd, APP Mesh, NGINX, Kuma, Gateway, Gloo
type Provider string

const (
	Istio Provider = "istio"
)

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
