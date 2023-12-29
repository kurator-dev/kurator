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
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,categories=kurator-dev
// +kubebuilder:subresource:status

// Pipeline is the top-level type for Kurator CI Pipeline.
type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineSpec   `json:"spec"`
	Status PipelineStatus `json:"status,omitempty"`
}

// PipelineSpec defines the desired state of a Pipeline.
type PipelineSpec struct {
	// Description allows an administrator to provide a description of the pipeline.
	// +optional
	Description string `json:"description,omitempty"`

	// Tasks is an ordered list of tasks in the pipeline, containing detailed information about each task.
	// The tasks will be executed in the order they are listed.
	Tasks []PipelineTask `json:"tasks"`

	// SharedWorkspace is the config of the PVC where task using
	// The PersistentVolumeClaim with this config will be created for each pipeline execution
	// it allows the user to specify e.g. size and StorageClass for the volume.
	// If not set, Kurator will create a PVC named Pipeline.name using default config
	// +optional
	SharedWorkspace *VolumeClaimTemplate `json:"sharedWorkspace,omitempty"`
}

// VolumeClaimTemplate is the configuration for the volume claim template in pipeline execution.
// For more details, see https://github.com/kubernetes/api/blob/master/core/v1/types.go
type VolumeClaimTemplate struct {
	// AccessMode determines the access modes for the volume, e.g., ReadWriteOnce.
	// This affects how the volume can be mounted.
	// "ReadWriteOnce" can be mounted in read/write mode to exactly 1 host
	// "ReadOnlyMany" can be mounted in read-only mode to many hosts
	// "ReadWriteMany" can be mounted in read/write mode to many hosts
	// "ReadWriteOncePod" can be mounted in read/write mode to exactly 1 pod, cannot be used in combination with other access modes
	AccessMode corev1.PersistentVolumeAccessMode `json:"accessMode,omitempty"`

	// StorageRequest defines the storage size required for this PVC, e.g., 1Gi, 100Mi.
	// It specifies the storage capacity needed as part of ResourceRequirements.
	// +kubebuilder:validation:Pattern="^[0-9]+(\\.[0-9]+)?(Gi|Mi)$"
	StorageRequest string `json:"requestsStorage,omitempty"`

	// StorageClassName specifies the StorageClass name to which this persistent volume belongs, e.g., manual.
	// It allows the PVC to use the characteristics defined by the StorageClass.
	StorageClassName string `json:"storageClassName,omitempty"`

	// VolumeMode specifies whether the volume should be used with a formatted filesystem (Filesystem)
	// or remain in raw block state (Block). The Filesystem value is implied when not included.
	// "Block"  means the volume will not be formatted with a filesystem and will remain a raw block device.
	// "Filesystem"  means the volume will be or is formatted with a filesystem.
	VolumeMode corev1.PersistentVolumeMode `json:"volumeMode,omitempty"`
}

type PipelineTask struct {
	// Name is the name of the task.
	Name string `json:"name"`

	// PredefinedTask allows users to select a predefined task.
	// Users can choose a predefined task from a set list and fill in their own parameters.
	// +optional
	PredefinedTask *PredefinedTask `json:"predefinedTask,omitempty"`

	// CustomTask enables defining a task directly within the CRD if TaskRef is not used.
	// This should only be used when TaskRef is not provided.
	// +optional
	CustomTask *CustomTask `json:"customTask,omitempty"`

	// Retries represents how many times this task should be retried in case of task failure.
	// default values is zero.
	// +optional
	Retries int `json:"retries,omitempty"`
}

type TaskTemplate string

const (
	// GitClone is typically the first task in the entire pipeline.
	// It clones the user's code repository into the workspace. This allows subsequent tasks to operate on this basis.
	// Since the pipeline is linked to a specific repository's webhook, Kurator automatically retrieves the repository information when triggered by the webhook.
	// Users don't need to configure additional repository information for this task, except for authentication details for private repositories.
	// This Predefined Task is origin from https://github.com/tektoncd/catalog/tree/main/task/git-clone/0.9
	// Here are the params that user can config:
	// - git-secret-name: The name of the secret for Git basic authentication.
	//   Kurator uses this Git credential to clone private repositories.
	//   The secret is formatted as follows, and more details can be found at:
	//   https://kurator.dev/docs/fleet-manager/pipeline/
	//   - username: <cleartext username>
	//   - password: <cleartext password>
	GitClone TaskTemplate = "git-clone"

	// GoTest runs Go tests in specified packages with configurable environment
	// This Predefined Task is origin from https://github.com/tektoncd/catalog/tree/main/task/golang-test/0.2/
	// Here are the params that user can config:
	// - packages: packages to test (default: ./...)
	// - context: path to the directory to use as context (default: .)
	// - version: golang version to use for builds (default: latest)
	// - flags: flags to use for go test command (default: -race -cover -v)
	// - GOOS: operating system target (default: linux)
	// - GOARCH: architecture target (default: amd64)
	// - GO111MODULE: value of module support (default: auto)
	// - GOCACHE: value for go caching path (default: "")
	// - GOMODCACHE: value for go module caching path (default: "")
	GoTest TaskTemplate = "go-test"

	// TODO: add more PredefinedTask
)

// PredefinedTask provides a structure for defining a PredefinedTask.
type PredefinedTask struct {
	// Name specifies the predefined task template to be used.
	// This field is required to select the appropriate PredefinedTask.
	// +required
	Name TaskTemplate `json:"name"`

	// Params contains key-value pairs for task-specific parameters.
	// The required parameters vary depending on the TaskType chosen.
	// +optional
	Params map[string]string `json:"params,omitempty"`
}

// CustomTask defines the specification for a user-defined task.
type CustomTask struct {
	// Image specifies the Docker image name.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	// +optional
	Image string `json:"image,omitempty"`

	// Command is the entrypoint array. It's not executed in a shell.
	// If not provided, the image's ENTRYPOINT is used.
	// Environment variables can be used in the format $(VAR_NAME).
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	// +listType=atomic
	Command []string `json:"command,omitempty"`

	// Args are the arguments for the entrypoint.
	// If not provided, the image's CMD is used.
	// Supports environment variable expansion in the format $(VAR_NAME).
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	// +listType=atomic
	Args []string `json:"args,omitempty"`

	// List of environment variables to set in the Step.
	// Cannot be updated.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=atomic
	Env []corev1.EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name"`

	// ResourceRequirements required by this Step.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +optional
	ResourceRequirements corev1.ResourceRequirements `json:"computeResources,omitempty"`

	// Script is the contents of an executable file to execute.
	// If Script is not empty, the CustomTask cannot have a Command and the Args will be passed to the Script.
	// +optional
	Script string `json:"script,omitempty"`
}

type PipelinePhase string

const (
	// RunningPhase indicates that the associated resources are currently being created.
	RunningPhase PipelinePhase = "Running"

	// FailedPhase signifies that the creation of associated resources has failed.
	FailedPhase PipelinePhase = "Failed"

	// ReadyPhase represents the state where all associated resources have been successfully created.
	ReadyPhase PipelinePhase = "Ready"
)

type PipelineStatus struct {
	// Phase describes the overall state of the Pipeline.
	// +optional
	Phase PipelinePhase `json:"phase,omitempty"`

	// EventListenerServiceName specifies the name of the service created by Kurator for event listeners.
	// This name is useful for users when setting up a gateway service and routing to this service.
	// +optional
	EventListenerServiceName *string `json:"eventListenerServiceName,omitempty"`
}

// PipelineList contains a list of Pipeline.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}
