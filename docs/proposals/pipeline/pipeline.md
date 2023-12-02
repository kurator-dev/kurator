---
title: Pipeline in Kurator
authors:
- "@Xieql"
reviewers:
approvers:

creation-date: 2023-11-22

---

## Pipeline in Kurator

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

### Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. 

A good summary is probably at least a paragraph in length.
-->

This proposal introduces a new feature to the Kurator project, aiming to simplify and streamline the process of setting up and managing CI/CD pipelines in cloud-native applications. 
Leveraging the capabilities of Tekton, this feature will offer a set of pre-configured, best-practice Pipeline task templates that users can deploy with a single click. 
Additionally, it will allow users to customize their own Tasks, offering both ease of use for beginners and flexibility for advanced users.

This feature is designed to encapsulate the complexity of CI/CD pipeline, making it more accessible to a wider range of users.

### Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users.
-->

The current landscape of CI/CD in cloud-native development often presents a steep learning curve due to the complexity of tools like Tekton. 

Users are required to have in-depth knowledge of various configurations and components to set up a functional pipeline. 

This proposal aims to eliminate these barriers, providing a user-friendly interface that simplifies the process, thereby accelerating development and deployment workflows in cloud-native projects.

#### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- To provide an easy-to-use interface for setting up and managing CI/CD pipelines in the Kurator project.

- To offer pre-configured Pipeline task templates based on best practices for various common CI/CD scenarios.

- To allow customization of Tasks, catering to the specific needs of the project.

- To improve the accessibility of cloud-native CI/CD practices to a broader audience, including those new to the field.

- To leverage Tekton Chains' capabilities to provide automated artifact build processes and signature addition, 
ensuring supply chain security and meeting the requirements of Software Supply Chain Level for Software Artifacts (SLSA).
  
- To expand the capabilities of the Kurator CLI, including querying the execution status and logs of Pipeline executions.



#### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- To completely replace the advanced functionalities and flexibility of Tekton. This project aims to simplify processes, not to substitute the core features of Tekton.

- To provide pre-configured solutions for every possible CI/CD scenario. While the aim is to cover common scenarios, some particularly complex or rare use cases may not be within the scope of this proposal.

- To offer comprehensive monitoring and troubleshooting solutions. Although basic execution status and log queries are provided, in-depth monitoring and troubleshooting may be beyond the scope of this proposal.

### Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

#### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

##### Story 1

A new user with minimal knowledge wants to set up a CI/CD pipeline for their cloud-native application. 
They select a pre-configured Pipeline template in Kurator, customize it minimally, and deploy it successfully with minimal effort.

##### Story 2

An experienced developer needs to customize their CI/CD pipeline with specific Tasks that are not covered in the pre-configured templates. 
They use Kurator to add their custom Tasks to the pipeline, benefiting from the combination of pre-configured and custom elements.

#### Notes/Constraints/Caveats (Optional)

- The feature will require ongoing maintenance to keep up with changes in Tekton, Kubernetes and task templates.

- User education and documentation will be key to successful adoption.

#### Risks and Mitigations

- Risk: Users may find the pre-configured templates too rigid for complex workflows.

- Mitigation: Provide clear documentation on how to customize and extend pipelines.

### Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

The design will focus on creating a user interface within Kurator that interacts seamlessly with Tekton's backend. 
The interface will provide options to select, customize, and deploy Pipeline templates. 
The pipeline controller will auto transform kurator pipeline into Tekton custom resources.

#### Overall Design

![use-pipeline](./image/use-pipeline.svg)


#### API Design

In this section, we delve into the detailed API designs for Pipeline

##### Pipeline API

Here's the preliminary design for Pipeline API:

```console

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

type TaskRef struct {
	// TaskType is used to specify the type of the predefined task.
	// This is a required field and determines which task template will be used.
	TaskType string `json:"taskType"`

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

const(
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
```


#### How to trigger pipeline

![trigger](./image/trigger.svg)

#### pipelineRun example

![pipelinerun-example](./image/pipelinerun-example.svg)

#### Supply Chain Security

##### SLSA Level 2 Key Requirements

1. **Verifiable Signatures**:

    SLSA Level 2 requires verifiable signatures associated with artifacts (like software packages, container images) to ensure their authenticity and integrity.

2. **Complete Build Process Records**:

    Detailed documentation of the entire build process, including inputs and outputs. These records, known as "provenance", provide transparency in artifact creation.

##### Role of Tekton Chains

1. **Generating Provenance**:

    Upon completion of a Pipeline, Tekton Chains automatically generates a detailed provenance for each `PipelineRun`. This includes key information like inputs and outputs of the build process.

2. **Automating Signature Creation**:

    Tekton Chains then generates a digital signature for this provenance, ensuring its authenticity and integrity, rather than directly signing the artifact.

##### User Verification

Users can verify the authenticity of the provenance and its signature before using an artifact, ensuring it was created through a verified build process, thus enhancing security.

For more details, see [Tekton signed-provenance-tutorial](https://tekton.dev/docs/chains/signed-provenance-tutorial/)

#### Kurator Pipeline cli

##### 1. Pipeline Commands

- **`pipeline list`**
  
    Display all current Pipelines.

- **`pipeline describe <name>`**

    Show detailed information of a specified Pipeline.

- **`pipeline log --lastExecution <name>`**

    Display the logs of the most recent PipelineExecution triggered by an event for a specified Pipeline.

##### 2. PipelineExecution Commands

- **`pipeline-execution list [--pipeline=<name>]`**
  
    List all PipelineExecution instances. The list includes each execution's event information and timestamp. 
    
    If the `--pipeline` parameter is provided, only list PipelineExecutions belonging to a specific Pipeline.

- **`pipeline-execution describe <name>`**

    Show detailed information of a specified PipelineExecution, including task composition, current status.

- **`pipeline-execution logs <name>`**

    Retrieve and display logs of a specified PipelineExecution.

#### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.
-->

End-to-End Tests: Comprehensive E2E tests should be conducted to ensure the pipeline processes work seamlessly across different clusters.

Integration Tests: Integration tests should be designed to ensure Kurator's functions as expected.

Unit Tests: Unit tests should cover the core functionalities and edge cases.

Isolation Testing: The pipeline functionalities should be tested in isolation and in conjunction with other components to ensure compatibility and performance.


### Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

Alternative: Integrating with Other CI/CD Tools

Consideration: Integrating with other existing CI/CD tools like Jenkins, GitLab CI, or GitHub Actions was also considered.

Rationale for Rejection: While these tools are powerful, they may not offer the same level of customization and Kubernetes-native capabilities as Tekton. 
Additionally, this approach would have required extensive modifications to align with the cloud-native focus of the Kurator project.

<!--
Note: This is a simplified version of kubernetes enhancement proposal template.
https://github.com/kubernetes/enhancements/tree/3317d4cb548c396a430d1c1ac6625226018adf6a/keps/NNNN-kep-template
-->
