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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pipelineapi "kurator.dev/kurator/pkg/apis/pipeline/v1alpha1"
)

const (
	TriggerTemplateName = "pipeline trigger template"
)

type TriggerConfig struct {
	PipelineName      string
	PipelineNamespace string
	OwnerReference    *metav1.OwnerReference
	AccessMode        string
	StorageRequest    string
	StorageClassName  string
	VolumeMode        string
}

// ServiceAccountName is the service account used by trigger
func (cfg TriggerConfig) ServiceAccountName() string {
	return cfg.PipelineName
}

// RenderTriggerWithPipeline takes a pipeline object and generates YAML byte array configuration representing the trigger configuration.
func RenderTriggerWithPipeline(pipeline *pipelineapi.Pipeline) ([]byte, error) {
	config := TriggerConfig{
		PipelineName:      pipeline.Name,
		PipelineNamespace: pipeline.Namespace,
		OwnerReference:    GeneratePipelineOwnerRef(pipeline),
	}
	if pipeline.Spec.SharedWorkspace != nil {
		config.AccessMode = string(pipeline.Spec.SharedWorkspace.AccessMode)
		config.StorageRequest = pipeline.Spec.SharedWorkspace.StorageRequest
		config.StorageClassName = pipeline.Spec.SharedWorkspace.StorageClassName
		config.VolumeMode = string(pipeline.Spec.SharedWorkspace.VolumeMode)
	}

	return RenderTrigger(config)
}

// RenderTrigger takes a TriggerConfig object and generates YAML byte array configuration representing the trigger configuration.
func RenderTrigger(cfg TriggerConfig) ([]byte, error) {
	return renderTemplate(TriggerTemplateContent, TriggerTemplateName, cfg)
}

const TriggerTemplateContent = `apiVersion: triggers.tekton.dev/v1alpha1
kind: TriggerTemplate
metadata:
  name: {{ .PipelineName }}-triggertemplate
  namespace: {{ .PipelineNamespace }}
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
spec:
  params:
  - name: gitrevision
    description: The git revision
  - name: gitrepositoryurl
    description: The git repository url
  - name: namespace
    description: The namespace to create the resources
  resourceTemplates:
  - apiVersion: tekton.dev/v1beta1
    kind: PipelineRun
    metadata:
      generateName: {{ .PipelineName }}-run-
      namespace: $(tt.params.namespace)
    spec:
      serviceAccountName: {{ .ServiceAccountName }}
      pipelineRef:
        name: {{ .PipelineName }}
      params:
      - name: revision
        value: $(tt.params.gitrevision)
      - name: repo-url
        value: $(tt.params.gitrepositoryurl)
      workspaces:
      - name: kurator-pipeline-shared-data # there only one pvc workspace in each pipeline, and the name is kurator-pipeline-shared-data
        volumeClaimTemplate:
          spec:
            accessModes:
              - {{ default "ReadWriteOnce" .AccessMode }}
            resources:
              requests:
                storage: {{ default "1Gi" .StorageRequest }}
{{- if .VolumeMode }}
            volumeMode: {{ .VolumeMode }}
{{- end }}
{{- if .StorageClassName }}
            storageClassName: {{ .StorageClassName }}
{{- end }}
      - name: git-credentials
        secret:
          secretName: git-credentials
      - name: docker-credentials
        secret:
          secretName: docker-credentials  # auth for task
---
apiVersion: triggers.tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: {{ .PipelineName }}-triggerbinding
  namespace: {{ .PipelineNamespace}}
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
spec:
  params:
  - name: gitrevision
    value: $(body.head_commit.id)
  - name: namespace
    value: {{ .PipelineNamespace}}
  - name: gitrepositoryurl
    value: "https://github.com/$(body.repository.full_name)"
---
apiVersion: triggers.tekton.dev/v1alpha1
kind: EventListener
metadata:
  name: {{ .PipelineName }}-listener
  namespace: {{ .PipelineNamespace}}
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
spec:
  serviceAccountName: {{ .ServiceAccountName }}
  triggers:
  - bindings:
    - ref: {{ .PipelineName }}-triggerbinding
    template:
      ref: {{ .PipelineName }}-triggertemplate
`
