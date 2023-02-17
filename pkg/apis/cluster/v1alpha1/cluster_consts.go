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
	// InfraProviderReadyCondition reports on wheter the infra is ready.
	InfraProviderReadyCondition capiv1.ConditionType = "InfraProviderReadyCondition"
	// InfraProviderProvisioningFailedReason (Severity=Error) documents that the infra provisioning failed.
	InfraProviderProvisioningFailedReason = "InfraProviderProvisioningFailedReason"

	CNIReadyCondition capiv1.ConditionType = "CNIReadyCondition"

	CNIProvisioningFailedReason = "CNIProvisioningFailedReason"

	ClusterAPIResourceProvisioningFailedReason = "ClusterAPIResourceProvisioningFailedReason"
)

// ClusterPhase is a string representation of the cluster's phase.
type ClusterPhase string

const (
	// ClusterPhaseProvisioning is the state when the cluster is being provisioned.
	ClusterPhaseProvisioning ClusterPhase = "Provisioning"

	// ClusterPhaseInfraProvisioned is the state when the infra has been provisioned.
	ClusterPhaseInfraProvisioned ClusterPhase = "IfraProvisioned"

	// ClusterCNIProvisioned is the state when the cni has been provisioned.
	ClusterCNIProvisioned ClusterPhase = "CNIProvisioned"

	// ClusterPhaseReady is the state when the cluster is ready.
	ClusterPhaseReady ClusterPhase = "Ready"

	// ClusterPhaseDeleting is the state when a delete request has been sent to the API Server.
	ClusterPhaseDeleting ClusterPhase = "Deleting"

	// ClusterPhaseFailed is the state when the cluster has failed to be provisioned.
	ClusterPhaseFailed ClusterPhase = "Failed"
)
