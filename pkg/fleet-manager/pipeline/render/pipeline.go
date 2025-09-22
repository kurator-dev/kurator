/*
Copyright 2022-2025 Kurator Authors.

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
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pipelineapi "kurator.dev/kurator/pkg/apis/pipeline/v1alpha1"
)

const (
	PipelineTemplateName       = "pipeline-template"
	DockerCredentialsName      = "dockerconfig"
	DockerCredentialsWorkspace = "docker-credentials"
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

type TaskInfo struct {
	Name       string
	TaskRef    string
	RunAfter   string
	Workspaces []Workspace
	Retries    int
}

type Workspace struct {
	Name      string
	Workspace string
}

func generateTasksInfo(pipelineName string, tasks []pipelineapi.PipelineTask) (dockerCredentialsWorkspace string, tasksInfo string, err error) {
	var tasksInfoBuilder strings.Builder
	tmpl, err := template.New("task").Parse(taskTemplate)
	if err != nil {
		return "", "", err
	}

	lastTask := string(pipelineapi.GitClone) // GitCloneTask is always the first task.
	for _, task := range tasks {
		if task.Name == string(pipelineapi.GitClone) {
			continue // Skip the first git-clone task because it is already fixed in template.
		}

		// Validate task
		if (task.CustomTask == nil && task.PredefinedTask == nil) || (task.CustomTask != nil && task.PredefinedTask != nil) {
			return "", "", fmt.Errorf("only one of 'PredefinedTask' or 'CustomTask' must be set in 'PipelineTask'")
		}

		taskInfo := TaskInfo{
			Name:     task.Name,
			TaskRef:  generatePipelineTaskName(task.Name, pipelineName),
			RunAfter: lastTask,
			Retries:  task.Retries,
			Workspaces: []Workspace{
				{Name: "source", Workspace: "kurator-pipeline-shared-data"},
			},
		}

		// Handle special cases
		if task.Name == string(pipelineapi.BuildPushImage) {
			taskInfo.Workspaces = append(taskInfo.Workspaces, Workspace{Name: DockerCredentialsName, Workspace: DockerCredentialsWorkspace})
			dockerCredentialsWorkspace = DockerCredentialsWorkspace
		}

		// Render task info using template
		if err := tmpl.Execute(&tasksInfoBuilder, taskInfo); err != nil {
			return "", "", err
		}

		lastTask = task.Name // Update the last task.
	}

	return dockerCredentialsWorkspace, tasksInfoBuilder.String(), nil
}

func generatePipelineTaskName(taskName, pipelineName string) string {
	return taskName + "-" + pipelineName
}

const taskTemplate = `  - name: {{.Name}}
    taskRef:
      name: {{.TaskRef}}
    runAfter: ["{{.RunAfter}}"]
    workspaces:
    {{- range .Workspaces}}
    - name: {{.Name}}
      workspace: {{.Workspace}}
    {{- end}}
    {{- if gt .Retries 0}}
    retries: {{.Retries}}
    {{- end}}
`

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
