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
	// CredentialsReadyCondition reports on wheter the credential of infra is ready.
	CredentialsReadyCondition capiv1.ConditionType = "ClusterCredentialsReady"
	// CredentialsProvisioningFailedReason (Severity=Error) documents that the cluster credentials provisioning failed.
	ClusterCredentialsProvisioningFailedReason = "ClusterCredentialsProvisioningFailed"
	// IAMProfileReadyCondition reports on wheter the IAM profile of infra is ready.
	IAMProfileReadyCondition capiv1.ConditionType = "IAMProfileReadyCondition"
	// IAMProfileProvisioningFailedReason (Severity=Error) documents that the cluster IAM profile provisioning failed.
	IAMProfileProvisioningFailedReason = "ClusterCredentialsProvisioningFailed"
	// ClusterAPIResourceReadyCondition reports on wheter the cluster API resource is ready.
	ClusterAPIResourceReadyCondition capiv1.ConditionType = "ClusterAPIResourceReady"
	// ClusterAPIResourceProvisioningFailedReason (Severity=Error) documents that the cluster API resource provisioning failed.
	ClusterAPIResourceProvisioningFailedReason = "ClusterAPIResourceProvisioningFailed"
)

// ClusterPhase is a string representation of the cluster's phase.
type ClusterPhase string

const (
	// ClusterPhasePending is the first state after the cluster is created.
	ClusterPhasePending ClusterPhase = "Pending"

	// ClusterPhaseProvisioning is the state when the cluster is being provisioned.
	ClusterPhaseProvisioning ClusterPhase = "Provisioning"

	// ClusterPhaseProvisioned is the state when the cluster has been provisioned.
	ClusterPhaseProvisioned ClusterPhase = "Provisioned"

	// ClusterPhaseReady is the state when the cluster is ready.
	ClusterPhaseReady ClusterPhase = "Ready"

	// ClusterPhaseDeleting is the state when a delete request has been sent to the API Server.
	ClusterPhaseDeleting ClusterPhase = "Deleting"

	// ClusterPhaseFailed is the state when the cluster has failed to be provisioned.
	ClusterPhaseFailed ClusterPhase = "Failed"
)
