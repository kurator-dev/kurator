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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pipelineapi "kurator.dev/kurator/pkg/apis/pipeline/v1alpha1"
)

const (
	CustomTaskTemplateName = "pipeline custom task template"
)

type CustomTaskConfig struct {
	TaskName             string
	PipelineName         string
	PipelineNamespace    string
	Image                string
	Command              []string
	Args                 []string
	Env                  []corev1.EnvVar
	ResourceRequirements *corev1.ResourceRequirements
	Script               string
	OwnerReference       *metav1.OwnerReference
}

// CustomTaskName is the name of custom task object, in case different pipeline have the same name task.
func (cfg CustomTaskConfig) CustomTaskName() string {
	return cfg.TaskName + "-" + cfg.PipelineName
}

// RenderCustomTaskWithPipeline takes a Pipeline object and generates YAML byte array configuration representing the CustomTask configuration.
func RenderCustomTaskWithPipeline(pipeline *pipelineapi.Pipeline, taskName string, task *pipelineapi.CustomTask) ([]byte, error) {
	cfg := CustomTaskConfig{
		TaskName:             taskName,
		PipelineName:         pipeline.Name,
		PipelineNamespace:    pipeline.Namespace,
		Image:                task.Image,
		Command:              task.Command,
		Args:                 task.Args,
		Env:                  task.Env,
		ResourceRequirements: &task.ResourceRequirements,
		Script:               task.Script,
		OwnerReference:       GeneratePipelineOwnerRef(pipeline),
	}

	return RenderCustomTask(cfg)
}

// RenderCustomTask takes a CustomTaskConfig object and generates YAML byte array configuration representing the CustomTask configuration.
func RenderCustomTask(cfg CustomTaskConfig) ([]byte, error) {
	if cfg.Image == "" || cfg.CustomTaskName() == "" {
		return nil, fmt.Errorf("invalid RBACConfig: PipelineName and PipelineNamespace must not be empty")
	}
	return renderTemplate(CustomTaskTemplateContent, CustomTaskTemplateName, cfg)
}

const CustomTaskTemplateContent = `apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: {{ .CustomTaskName }}
  namespace: {{ .PipelineNamespace }}
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
spec:
  description: >-
    This task is a user-custom, single-step task.
    The workspace is automatically and exclusively created named "source",
    and assigned to the workspace of the pipeline in which this task is located.
  workspaces:
  - name: source
    description: The workspace where user to run user-custom task.
  steps:
  - name: {{ .CustomTaskName }}
    image: {{ .Image }}
    {{- if .Env }}
    env:
    {{- range .Env }}
    - name: {{ .Name }}
      value: {{ .Value }}
    {{- end }}
    {{- end }}
    {{- if .Command }}
    command:
    {{- range .Command }}
    - {{ . }}
    {{- end }}
    {{- end }}
    {{- if .Args }}
    args:
    {{- range .Args }}
    - {{ . }}
    {{- end }}
    {{- end }}
    {{- if .Script }}
    script: |
      {{ .Script }}
    {{- end }}
    {{- if .ResourceRequirements }}
    resources:
      {{- if .ResourceRequirements.Requests }}
      requests:
        {{- if .ResourceRequirements.Requests.Cpu }}
        cpu: {{ .ResourceRequirements.Requests.Cpu }}
        {{- end }}
        {{- if .ResourceRequirements.Requests.Memory }}
        memory: {{ .ResourceRequirements.Requests.Memory }}
        {{- end }}
      {{- end }}
      {{- if .ResourceRequirements.Limits }}
      limits:
        {{- if .ResourceRequirements.Limits.Cpu }}
        cpu: {{ .ResourceRequirements.Limits.Cpu }}
        {{- end }}
        {{- if .ResourceRequirements.Limits.Memory }}
        memory: {{ .ResourceRequirements.Limits.Memory }}
        {{- end }}
      {{- end }}
    {{- end }}
`
