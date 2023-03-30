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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=customclusters,shortName=cc
// +kubebuilder:subresource:status

// CustomCluster represents the parameters for a cluster in supplement of Cluster API.
type CustomCluster struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the kurator cluster.
	Spec CustomClusterSpec `json:"spec,omitempty"`

	// Current status of the kurator cluster
	Status CustomClusterStatus `json:"status,omitempty"`
}

// CustomClusterSpec defines the desired state of a kurator cluster.
type CustomClusterSpec struct {
	// TODO: any other configurations that does not exist in upstream cluster api

	// MachineRef is the reference of nodes for provisioning a kurator cluster.
	// +optional
	MachineRef *corev1.ObjectReference `json:"machineRef,omitempty"`

	// CNIConfig is the configuration for the CNI of the cluster.
	CNI CNIConfig `json:"cni"`
}

type CNIConfig struct {
	// Type is the type of CNI. The default value is calico and can be ["calico", "cilium", "canal", "flannel"]
	Type string `json:"type"`
}

type CustomClusterPhase string

const (
	// PendingPhase represents the customCluster's first phase after being created
	PendingPhase CustomClusterPhase = "Pending"

	// ProvisioningPhase represents the cluster is provisioning. In this phase, the worker named ends in "init" is running to initialize the cluster
	ProvisioningPhase CustomClusterPhase = "Provisioning"

	// ProvisionedPhase represents the cluster has been created and configured. In this phase, the worker named ends in "init" is completed
	ProvisionedPhase CustomClusterPhase = "Provisioned"

	// DeletingPhase represents the delete request has been sent but cluster on has not yet been completely deleted. In this phase, the worker named ends in "terminate" is running to clear the cluster.
	DeletingPhase CustomClusterPhase = "Deleting"

	// ProvisionFailedPhase represents something is wrong when creating the cluster. In this phase, the worker named ends in "init" is in error
	ProvisionFailedPhase CustomClusterPhase = "ProvisionFailed"

	// UnknownPhase represents provisioned cluster state cannot be determined. It can be scaling failed or deleting failed.
	UnknownPhase CustomClusterPhase = "Unknown"

	// ScalingUpPhase represents the cluster is adding the worker nodes.
	ScalingUpPhase CustomClusterPhase = "ScalingUp"

	// ScalingDownPhase represents the cluster is removing the worker nodes.
	ScalingDownPhase CustomClusterPhase = "ScalingDown"
)

const (
	// ReadyCondition reports on whether the cluster is provisioned.
	ReadyCondition capiv1.ConditionType = "Ready"
	// FailedCreateInitWorker (Severity=Error) documents that the initialization worker failed to create.
	FailedCreateInitWorker = "InitWorkerFailedCreate"
	// InitWorkerRunFailedReason (Severity=Error) documents that the initialization worker run failed.
	InitWorkerRunFailedReason = "InitWorkerRunFailed"

	// ScaledUpCondition reports on whether the cluster worker nodes is scaled up.
	ScaledUpCondition capiv1.ConditionType = "ScaledUp"
	// FailedCreateScaleUpWorker (Severity=Error) documents that the scale up worker failed to create.
	FailedCreateScaleUpWorker = "ScaleUpWorkerFailedCreate"
	// ScaleUpWorkerRunFailedReason (Severity=Error) documents that the scale up worker run failed.
	ScaleUpWorkerRunFailedReason = "ScaleUpWorkerRunFailed"

	// ScaledDownCondition reports on whether the cluster worker nodes is scaled down.
	ScaledDownCondition capiv1.ConditionType = "ScaledDown"
	// FailedCreateScaleDownWorker (Severity=Error) documents that the scale down worker failed to create.
	FailedCreateScaleDownWorker = "ScaleDownWorkerFailedCreate"
	// ScaleDownWorkerRunFailedReason (Severity=Error) documents that the scale down worker run failed.
	ScaleDownWorkerRunFailedReason = "ScaleDownWorkerRunFailed"

	// TerminatedCondition reports on whether the cluster is terminated. If this condition meet, then the customCluster will be deleted and there won't be any marking as true.
	TerminatedCondition capiv1.ConditionType = "Terminated"
	// FailedCreateTerminateWorker (Severity=Error) documents that the terminal worker failed to create.
	FailedCreateTerminateWorker = "TerminateWorkerFailedCreate"
	// TerminateWorkerRunFailedReason (Severity=Error) documents that the terminal worker run failed.
	TerminateWorkerRunFailedReason = "TerminateWorkerRunFailed"
)

// CustomClusterStatus represents the current status of the cluster.
type CustomClusterStatus struct {
	// Conditions defines current service state of the cluster.
	// +optional
	Conditions capiv1.Conditions `json:"conditions,omitempty"`

	// Phase represents the current phase of customCluster actuation.
	// E.g.  Running, Succeed, Terminating, Failed etc.
	// +optional
	Phase CustomClusterPhase `json:"phase,omitempty"`

	// APIEndpoint is the endpoint to communicate with the apiserver.
	// Format should be: `https://host:port`
	// +optional
	APIEndpoint string `json:"apiEndpoint,omitempty"`

	// KubeconfigSecretRef represents the secret that contains the credential to access this cluster.
	// +optional
	KubeconfigSecretRef string `json:"kubeconfigSecretRef,omitempty"`
}

func (cc *CustomCluster) GetConditions() capiv1.Conditions {
	return cc.Status.Conditions
}

func (cc *CustomCluster) SetConditions(conditions capiv1.Conditions) {
	cc.Status.Conditions = conditions
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CustomClusterList contains a list of CustomCluster.
type CustomClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CustomCluster `json:"items"`
}
