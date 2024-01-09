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

package render

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pipelineapi "kurator.dev/kurator/pkg/apis/pipeline/v1alpha1"
)

const (
	PipelineTemplateName  = "pipeline template"
	DockerCredentialsName = "docker-credentials"
)

// PipelineConfig defines the configuration needed to render a pipeline.
type PipelineConfig struct {
	PipelineName      string
	PipelineNamespace string

	// TasksInfo contains the necessary information to integrate tasks into the pipeline.
	TasksInfo      string
	OwnerReference *metav1.OwnerReference

	// DockerCredentials is the name of Docker credentials secret for image build tasks.
	DockerCredentials string
}

// RenderPipelineWithPipeline renders the pipeline configuration as a YAML byte array.
// It uses the specified pipeline to gather necessary information.
func RenderPipelineWithPipeline(pipeline *pipelineapi.Pipeline) ([]byte, error) {
	dockerCredentials, tasksInfo, err := generateTasksInfo(pipeline.Name, pipeline.Spec.Tasks)
	if err != nil {
		return nil, err
	}

	cfg := PipelineConfig{
		PipelineName:      pipeline.Name,
		PipelineNamespace: pipeline.Namespace,
		TasksInfo:         tasksInfo,
		OwnerReference:    GeneratePipelineOwnerRef(pipeline),
		DockerCredentials: dockerCredentials,
	}

	return renderPipeline(cfg)
}

// renderPipeline creates a YAML representation of the pipeline configuration.
func renderPipeline(cfg PipelineConfig) ([]byte, error) {
	return renderTemplate(PipelineTemplateContent, PipelineTemplateName, cfg)
}

// generateTasksInfo creates a string representation of tasks for inclusion in a pipeline.
// It returns the name of Docker credentials if required by the tasks.
func generateTasksInfo(pipelineName string, tasks []pipelineapi.PipelineTask) (dockerCredentials string, tasksInfo string, err error) {
	var tasksBuilder strings.Builder

	lastTask := "git-clone" // GitCloneTask is always the first task.
	for _, task := range tasks {
		if task.Name == "git-clone" {
			continue // Skip the first git-clone task.because it is already fixed in template.
		}

		var taskInfo string
		if (task.CustomTask == nil && task.PredefinedTask == nil) || (task.CustomTask != nil && task.PredefinedTask != nil) {
			return "", "", fmt.Errorf("only one of 'PredefinedTask' or 'CustomTask' must be set in 'PipelineTask'")
		}

		if task.Name == "build-and-push-image" { // build-and-push-image need special handle.
			taskInfo = generateKanikoTaskInfo(task.Name, generatePipelineTaskName(task.Name, pipelineName), lastTask, task.Retries)
			dockerCredentials = DockerCredentialsName
		} else {
			taskInfo = generateTaskInfo(task.Name, generatePipelineTaskName(task.Name, pipelineName), lastTask, task.Retries)
		}

		fmt.Fprintf(&tasksBuilder, "  %s", taskInfo)
		lastTask = task.Name // Update the last task.
	}

	return dockerCredentials, tasksBuilder.String(), nil
}

// generateTaskInfo formats a single task's information.
func generateTaskInfo(taskName, taskRefer, lastTask string, retries int) string {
	return generateTaskInfoBase(taskName, taskRefer, lastTask, retries)
}

// generateKanikoTaskInfo formats Kaniko task information, including additional Docker credentials workspace.
func generateKanikoTaskInfo(taskName, taskRefer, lastTask string, retries int) string {
	dockerWorkspace := "    - name: dockerconfig\n      workspace: docker-credentials\n"
	return generateTaskInfoBase(taskName, taskRefer, lastTask, retries, dockerWorkspace)
}

// generateTaskInfoBase formats the base information of a task.
func generateTaskInfoBase(taskName, taskRefer, lastTask string, retries int, additionalWorkspaces ...string) string {
	var taskBuilder strings.Builder

	// Define task name and reference
	fmt.Fprintf(&taskBuilder, "- name: %s\n    taskRef:\n      name: %s\n", taskName, taskRefer)

	// Specify dependency on the preceding task
	fmt.Fprintf(&taskBuilder, "    runAfter: [\"%s\"]\n", lastTask)

	// Add fixed workspace configuration
	taskBuilder.WriteString("    workspaces:\n    - name: source\n      workspace: kurator-pipeline-shared-data\n")

	// Add additional workspaces if any additionalWorkspaces exist
	for _, workspace := range additionalWorkspaces {
		taskBuilder.WriteString(workspace)
	}

	// Include retry configuration if applicable
	if retries > 0 {
		fmt.Fprintf(&taskBuilder, "    retries: %d\n", retries)
	}

	return taskBuilder.String()
}

func generatePipelineTaskName(taskName, pipelineName string) string {
	return taskName + "-" + pipelineName
}

const PipelineTemplateContent = `apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: {{ .PipelineName}}
  namespace: {{ .PipelineNamespace }}
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
spec:
  description: |
    This is a universal pipeline with the following settings: 
      1. No parameters are passed because all user parameters have already been rendered into the corresponding tasks. 
      2. All tasks are strictly executed in the order defined by the user, with each task starting only after the previous one is completed. 
      3. There is only one workspace, which is used by all tasks. The PVC for this workspace will be configured in the trigger.
  params:
  - name: repo-url
    type: string
    description: The git repository URL to clone from.
  - name: revision
    type: string
    description: The git branch to clone.
  workspaces:
  - name: kurator-pipeline-shared-data
    description: |
      This workspace is used by all tasks
  - name: git-credentials
    description: |
      A Workspace containing a .gitconfig and .git-credentials file. These
      will be copied to the user's home before any git commands are run. Any
      other files in this Workspace are ignored.
{{- if .DockerCredentials }}
  - name: docker-credentials
    description: |
      This is the credentials for build and push image task.
{{- end }}
  tasks:
  - name: git-clone
    # Key points about 'git-clone':
    # - Fundamental for all tasks.
    # - Closely integrated with the trigger.
    # - Always the first task in the pipeline.
    # - Cannot be modified via templates.
    taskRef:
      name: git-clone-{{ .PipelineName }}
    workspaces:
    - name: source
      workspace: kurator-pipeline-shared-data
    - name: basic-auth
      workspace: git-credentials
    params:
    - name: url
      value: $(params.repo-url)
    - name: revision
      value: $(params.revision)
{{ .TasksInfo }}`
