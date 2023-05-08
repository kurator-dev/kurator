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
	"k8s.io/apimachinery/pkg/util/intstr"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,categories=kurator-dev
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="InfraType",type="string",JSONPath=".spec.infraType",description="Infra type of the cluster"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version",description="Kubernetes version of the cluster"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Phase of the cluster"

// Cluster is the schema for the cluster's API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ClusterSpec   `json:"spec,omitempty"`
	Status            ClusterStatus `json:"status,omitempty"`
}

// ClusterSpec defines the desired state of the Cluster
type ClusterSpec struct {
	// InfraType is the infra type of the cluster.
	InfraType ClusterInfraType `json:"infraType"`
	// Credential is the credential used to access the cloud provider.
	// +optional
	Credential *CredentialConfig `json:"credential,omitempty"`
	// Version is the Kubernetes version to use for the cluster.
	Version string `json:"version"`
	// Region is the region to deploy the cluster.
	Region string `json:"region"`
	// Network is the network configuration for the cluster.
	Network NetworkConfig `json:"network"`
	// Master is the configuration for the master node.
	Master MasterConfig `json:"master"`
	// Workers is the list of worker nodes.
	Workers []WorkerConfig `json:"workers"`
	// PodIdentity is the configuration for the pod identity.
	// +optional
	PodIdentity PodIdentityConfig `json:"podIdentity,omitempty"`
	// AdditionalResources provides a way to automatically apply a set of resouces to cluster after it's ready.
	// Note: the resouces will only apply once.
	// +optional
	AdditionalResources []ResourceRef `json:"additionalResources,omitempty"`
}

type ClusterInfraType string

const (
	// AWSClusterInfraType is the type for the cluster on AWS infra.
	AWSClusterInfraType ClusterInfraType = "aws"
)

type CredentialConfig struct {
	SecretRef string `json:"secretRef"`
}

type NetworkConfig struct {
	// VPC is the configuration for the VPC.
	VPC VPCConfig `json:"vpc"`
	// PodCIDRs is the CIDR block for pods in this cluster.
	// Defaults to 192.168.0.0/16.
	// +optional
	// +kubebuilder:default:={"192.168.0.0/16"}
	PodCIDRs CIDRBlocks `json:"podCIDRs,omitempty"`
	// ServiceCIDRs is the CIDR block for services in this cluster.
	// Defaults to 10.96.0.0/12.
	// +optional
	// +kubebuilder:default:={"10.96.0.0/12"}
	ServiceCIDRs CIDRBlocks `json:"serviceCIDRs,omitempty"`
	// CNI is the configuration for the CNI.
	CNI CNIConfig `json:"cni"`
}

type CIDRBlocks []string

type VPCConfig struct {
	// ID defines a unique identifier to reference this resource.
	// +optional
	ID string `json:"id"`
	// Name is the name of the VPC.
	// if not set, the name will be generated from cluster name.
	// +optional
	Name string `json:"name,omitempty"`
	// CIDRBlock is the CIDR block to be used when the provider creates a managed VPC.
	// Defaults to 10.0.0.0/16.
	// +optional
	// +kubebuilder:default:="10.0.0.0/16"
	CIDRBlock string `json:"cidrBlock"`
}

type CNIConfig struct {
	// Type is the type of CNI.
	Type string `json:"type"`
	// ExtraArgs is the set of extra arguments for CNI.
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}

type MasterConfig struct {
	NodeConfig `json:",inline"`
}

type WorkerConfig struct {
	NodeConfig `json:",inline"`
	// Strategy to use to replace existing nodes with new ones.
	// +optional
	Strategy *NodeUpgradeStrategy `json:"strategy,omitempty"`
}

type NodeUpgradeStrategy struct {
	// Type of node replacement strategy.
	// Default is RollingUpdate.
	// +optional
	Type NodeUpgradeStrategyType `json:"type,omitempty"`
	// RollingUpdate config params. Present only if NodeUpgradeStrategyType = RollingUpdate.
	// +optional
	RollingUpdate *RollingUpdateNodeUpgradeStrategy `json:"rollingUpdate,omitempty"`
}

type NodeUpgradeStrategyType string

const (
	// RollingUpdateNodeUpgradeStrategyType replaces old machines by new one using rolling update.
	RollingUpdateNodeUpgradeStrategyType NodeUpgradeStrategyType = "RollingUpdate"
	// OnDeleteNodeUpgradeStrategyType replaces old machines when the deletion of the asssoicated machines are completed.
	OnDeleteNodeUpgradeStrategyType NodeUpgradeStrategyType = "OnDelete"
)

type RollingUpdateNodeUpgradeStrategy struct {
	// MaxUnavailable is the maximum number of nodes that can be unavailable during the update.
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
	// MaxSurge is the maximum number of nodes that can be created above the desired number of nodes during the update.
	MaxSurge *intstr.IntOrString `json:"maxSurge,omitempty"`
	// DeletePolicy defines the policy used to identify nodes to delete when downscaling.
	// Valid values are "Random", "Newest" and "Oldest".
	// Defaults to "Newest".
	// +kubebuilder:validation:Enum=Random;Newest;Oldest
	// +optional
	DeletePolicy string `json:"deletePolicy,omitempty"`
}

// NodeConfig defines the configuration for the node.
type NodeConfig struct {
	MachineConfig          `json:",inline"`
	NodeRegistrationConfig `json:",inline"`
}

// MachineConfig defines the configuration for the machine.
type MachineConfig struct {
	// Replicas is the number of replicas of the machine.
	Replicas int `json:"replicas"`
	// InstanceType is the type of instance to use for the instance.
	InstanceType string `json:"instanceType"`
	// SSHKeyName is the name of the SSH key to use for the instance.
	// +optional
	SSHKeyName string `json:"sshKeyName,omitempty"`
	// ImageOS is the OS of the image to use for the instance.
	// Defaults to "ubuntu-20.04".
	// +optional
	// +kubebuilder:default:="ubuntu-20.04"
	ImageOS string `json:"imageOS,omitempty"`
	// RootVolume is the root volume to attach to the instance.
	// +optional
	RootVolume *Volume `json:"rootVolumeSize,omitempty"`
	// NonRootVolumes is the list of non-root volumes to attach to the instance.
	// +optional
	NonRootVolumes []Volume `json:"nonRootVolumes,omitempty"`
	// ExtraArgs is the set of extra arguments to create Machine on different infra.
	// +optional
	ExtraArgs apiextensionsv1.JSON `json:"extraArgs,omitempty"`
}

// NodeRegistrationConfig defines the configuration for the node registration.
type NodeRegistrationConfig struct {
	// Labels is the set of labels to apply to the nodes.
	Labels map[string]string `json:"labels,omitempty"`
	// Taints is the set of taints to apply to the nodes.
	Taints []corev1.Taint `json:"taints,omitempty"`
}

type Volume struct {
	// Type is the type of the volume (e.g. gp2, io1, etc...).
	// +optional
	Type string `json:"type"`
	// Size specifies size (in Gi) of the storage device.
	// Must be greater than the image snapshot size or 8 (whichever is greater).
	// +kubebuilder:validation:Minimum=8
	Size int64 `json:"size"`
}

// PodIdentityConfig defines the configuration for the pod identity.
type PodIdentityConfig struct {
	// Enabled is true when the pod identity is enabled.
	Enabled bool `json:"enabled"`
}

type ResourceRef struct {
	// Name is the name of the resource.
	// +kubectl:validation:MinLength=1
	Name string `json:"name"`
	// Kind Of the resource. e.g. ConfigMap, Secret, etc.
	// +kubebuilder:validation:Enum=ConfigMap;Secret
	Kind string `json:"kind"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// Conditions defines current service state of the cluster.
	// +optional
	Conditions capiv1beta1.Conditions `json:"conditions,omitempty"`
	// Phase is the current lifecycle phase of the cluster.
	// +optional
	Phase string `json:"phase,omitempty"`
	// APIEndpoint is the endpoint to communicate with the apiserver.
	// Format should be: `https://host:port`
	// +optional
	APIEndpoint string `json:"apiEndpoint,omitempty"`
	// KubeconfigSecretRef represents the secret that contains the credential to access this cluster.
	// +optional
	KubeconfigSecretRef string `json:"kubeconfigSecretRef,omitempty"`
	// ServiceAccountIssuer is the URL of the service account issuer.
	// +optional
	ServiceAccountIssuer string `json:"serviceAccountIssuer"`
	// Accepted indicates whether the cluster is registered to kurator fleet.
	Accepted bool `json:"accepted"`
}

func (c *Cluster) GetConditions() capiv1beta1.Conditions {
	return c.Status.Conditions
}

func (c *Cluster) SetConditions(conditions capiv1beta1.Conditions) {
	c.Status.Conditions = conditions
}

// ClusterList contains a list of Cluster.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func (c *Cluster) IsReady() bool {
	return c.Status.Phase == string(ClusterPhaseReady)
}

func (c *Cluster) GetObject() client.Object {
	return c
}

func (c *Cluster) GetSecretName() string {
	return c.Spec.Credential.SecretRef
}

// ClusterKubeconfigDataName is the key used to store a Kubeconfig in the secret's data field.
// This is derived from cluster api
const ClusterKubeconfigDataName = "value"

func (c *Cluster) GetSecretKey() string {
	return ClusterKubeconfigDataName
}

func (c *Cluster) SetAccepted(accepted bool) {
	c.Status.Accepted = accepted
}
