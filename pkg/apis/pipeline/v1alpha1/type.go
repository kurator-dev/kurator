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

	// SharedWorkspace is the name of the PVC. If not specified, a PVC with the Pipeline's name as prefix will be created by default.
	// If not set, Kurator will create a PVC named Pipeline.name using default config
	// +optional
	SharedWorkspace *string `json:"sharedWorkspace,omitempty"`
}

type PipelineTask struct {
	// Name is the name of the task.
	Name string `json:"name"`

	// TaskRef is a reference to a predefined task template.
	// Users should provide a TaskRef name from a predefined library.
	// +optional
	TaskRef *TaskRef `json:"taskRef,omitempty"`

	// CustomTask enables defining a task directly within the CRD if TaskRef is not used.
	// This should only be used when TaskRef is not provided.
	// +optional
	CustomTask *CustomTask `json:"customTask,omitempty"`

	// Retries represents how many times this task should be retried in case of task failure.
	// +optional
	Retries *int `json:"retries,omitempty"`
}

type PredefinedTask string

const (
	// GitClone is typically the first task in the entire pipeline.
	// It clones the user's code repository into the workspace. This allows subsequent tasks to operate on this basis.
	// Since the pipeline is linked to a specific repository's webhook, Kurator automatically retrieves the repository information when triggered by the webhook.
	// Users don't need to configure additional repository information for this task, except for authentication details for private repositories.
	// This Predefined Task is origin from https://github.com/tektoncd/catalog/tree/main/task/git-clone/0.9
	// Here are the params that user can config:
	// - git-secret-name: the secret name of git basic auth, Kurator use this git credential to clone private repo.
	GitClone PredefinedTask = "git-clone"

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
	GoTest PredefinedTask = "go-test"

	// TODO: add more PredefinedTask
)

type TaskRef struct {
	// TaskType is used to specify the type of the predefined task.
	// This is a required field and determines which task template will be used.
	TaskType PredefinedTask `json:"taskType"`

	// Params are key-value pairs of parameters for the predefined task.
	// These parameters depend on the selected task template.
	// +optional
	Params map[string]string `json:"params,omitempty"`
}

// CustomTask defines the specification for a user-defined task.
type CustomTask struct {
	// Image specifies the Docker image name.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	// +optional
	Image string `json:"image,omitempty" protobuf:"bytes,2,opt,name=image"`

	// Command is the entrypoint array. It's not executed in a shell.
	// If not provided, the image's ENTRYPOINT is used.
	// Environment variables can be used in the format $(VAR_NAME).
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	// +listType=atomic
	Command []string `json:"command,omitempty" protobuf:"bytes,3,rep,name=command"`

	// Args are the arguments for the entrypoint.
	// If not provided, the image's CMD is used.
	// Supports environment variable expansion in the format $(VAR_NAME).
	// More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell
	// +optional
	// +listType=atomic
	Args []string `json:"args,omitempty" protobuf:"bytes,4,rep,name=args"`

	// Step's working directory.
	// If not specified, the container runtime's default will be used, which
	// might be configured in the container image.
	// Cannot be updated.
	// +optional
	WorkingDir string `json:"workingDir,omitempty" protobuf:"bytes,5,opt,name=workingDir"`

	// List of environment variables to set in the Step.
	// Cannot be updated.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=atomic
	Env []corev1.EnvVar `json:"env,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,7,rep,name=env"`

	// ResourceRequirements required by this Step.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +optional
	ResourceRequirements corev1.ResourceRequirements `json:"computeResources,omitempty" protobuf:"bytes,8,opt,name=computeResources"`

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
