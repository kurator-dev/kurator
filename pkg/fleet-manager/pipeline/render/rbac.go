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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RBACTemplateName = "pipeline rbac template"
)

// RBACConfig contains the configuration data required for the RBAC template.
// Both PipelineName and PipelineNamespace are required.
type RBACConfig struct {
	PipelineName         string // Name of the pipeline.
	PipelineNamespace    string // Kubernetes namespace where the pipeline is deployed.
	OwnerReference       *metav1.OwnerReference
	ChainCredentialsName string
}

// ServiceAccountName generates the service account name using the pipeline name.
func (rbac RBACConfig) ServiceAccountName() string {
	return rbac.PipelineName
}

// RenderRBAC renders the RBAC configuration using a specified template.
func RenderRBAC(cfg RBACConfig) ([]byte, error) {
	if cfg.PipelineName == "" || cfg.PipelineNamespace == "" {
		return nil, fmt.Errorf("invalid RBACConfig: PipelineName and PipelineNamespace must not be empty")
	}
	return renderTemplate(RBACTemplateContent, RBACTemplateName, cfg)
}

const RBACTemplateContent = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: "{{ .ServiceAccountName }}"
  namespace: "{{ .PipelineNamespace }}"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
{{- if .ChainCredentialsName }}
secrets:
  - name: "chain-credentials"
    namespace: "{{ .PipelineNamespace }}"
{{- end }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: "{{ .PipelineName }}"
  namespace: "{{ .PipelineNamespace }}"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
subjects:
- kind: ServiceAccount
  name: "{{ .ServiceAccountName }}"
  namespace: "{{ .PipelineNamespace }}"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tekton-triggers-eventlistener-roles # add role for handle broad-resource, such as eventListener, triggers, configmaps and so on. tekton-triggers-eventlistener-roles is provided by Tekton
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: "{{ .PipelineName }}"
  namespace: "{{ .PipelineNamespace }}"
{{- if .OwnerReference }}
  ownerReferences:
  - apiVersion: "{{ .OwnerReference.APIVersion }}"
    kind: "{{ .OwnerReference.Kind }}"
    name: "{{ .OwnerReference.Name }}"
    uid: "{{ .OwnerReference.UID }}"
{{- end }}
subjects:
- kind: ServiceAccount
  name: "{{ .ServiceAccountName }}"
  namespace: "{{ .PipelineNamespace }}"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: tekton-triggers-eventlistener-clusterroles # add role for handle secret, clustertriggerbinding and clusterinterceptors. tekton-triggers-eventlistener-clusterroles is provided by Tekton
`
