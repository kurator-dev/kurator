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
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"sigs.k8s.io/yaml"
)

// renderTemplate reads, parses, and renders a template file using the provided configuration data.
func renderTemplate(tplFileString, tplName string, cfg interface{}) ([]byte, error) {
	tpl, err := template.New(tplName).Funcs(funMap()).Parse(tplFileString)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer

	if err := tpl.Execute(&b, cfg); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// funMap returns a map of functions for use in the template.
func funMap() template.FuncMap {
	m := sprig.TxtFuncMap()
	m["toYaml"] = toYaml
	return m
}

// toYaml converts a given value to its YAML representation.
func toYaml(value interface{}) string {
	y, err := yaml.Marshal(value)
	if err != nil {
		return ""
	}

	return string(y)
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
secrets:
  - name: "chain-credentials"
    namespace: "{{ .PipelineNamespace }}"
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: "{{ .BroadResourceRoleBindingName }}"
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
  name: "{{ .SecretRoleBindingName }}"
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
