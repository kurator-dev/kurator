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
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// InfrastructureReadyCondition reports a summary status of the infrastructure resources.
	InfrastructureReadyCondition capiv1.ConditionType = "InfrastructureReady"
	// InfrastructureProvisionFailedReason (Severity=Error) documents that the infrastructure provisioning failed.
	InfrastructureProvisionFailedReason = "InfrastructureProvisionFailed"
	// InfrastructureNotReadyReason (Severity=Error) documents that the infrastructure is not ready.
	InfrastructureNotReadyReason = "InfrastructureNotReady"

	//	CNICondition reports on whether the CNI is provisioned.
	CNICondition capiv1.ConditionType = "CNI"
	// CNIProvisionFailedReason (Severity=Error) documents that the CNI provisioning failed.
	CNIProvisionFailedReason = "CNIProvisionFailed"
	// CNIProvisionReadyReason (Severity=Error) documents that the CNI is not ready.
	CNINotReadyReason = "CNINotReady"

	// ReadyCondition defines the Ready condition type that summarizes the operational state of a Cluster.
	ReadyCondition capiv1.ConditionType = "Ready"
	// ProviderInitializeFailedReason (Severity=Error) documents that the cluster infrastructure provider initilize failed, probabily a wrong credential.
	ProviderInitializeFailedReason = "ProviderInitializeFailed"
	// ClusterResourceSetProvisionFailedReason (Severity=Error) documents that the additinal Cluster API resources (ClusterResourceSet etc.) provisioning failed.
	ClusterResourceSetProvisionFailedReason = "ClusterResourceSetProvisionFailed"
)

// ClusterPhase is a string representation of the cluster's phase.
type ClusterPhase string

const (
	// ClusterPhaseProvisioning is the state when the cluster is being provisioned.
	ClusterPhaseProvisioning ClusterPhase = "Provisioning"

	// ClusterPhaseReady is the state when the cluster is ready.
	// Ready means both cluster and CNI has been provisioned
	ClusterPhaseReady ClusterPhase = "Ready"

	// ClusterPhaseDeleting is the state when a delete request has been sent to the API Server.
	ClusterPhaseDeleting ClusterPhase = "Deleting"

	// ClusterPhaseFailed is the state when the cluster has failed to be provisioned.
	ClusterPhaseFailed ClusterPhase = "Failed"
)
