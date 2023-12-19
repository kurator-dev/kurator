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
	"io/fs"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// RBACTemplateFileName is the name of the RBAC template file.
	RBACTemplateFileName = "rbac/rbac.tpl"
	RBACTemplateName     = "pipeline rbac template"
	SecretSuffix         = "-secret"
	BroadResourceSuffix  = "-broad-resource"
)

// RBACConfig contains the configuration data required for the RBAC template.
// Both PipelineName and PipelineNamespace are required.
type RBACConfig struct {
	PipelineName      string // Name of the pipeline.
	PipelineNamespace string // Kubernetes namespace where the pipeline is deployed.
	OwnerReference    *metav1.OwnerReference
}

// ServiceAccountName generates the service account name using the pipeline name \
func (rbac RBACConfig) ServiceAccountName() string {
	return rbac.PipelineName
}

// BroadResourceRoleBindingName generates the role binding name using the service account name.
func (rbac RBACConfig) BroadResourceRoleBindingName() string {
	return rbac.ServiceAccountName() + BroadResourceSuffix
}

// SecretRoleBindingName generates the cluster role binding name using the service account name.
func (rbac RBACConfig) SecretRoleBindingName() string {
	return rbac.ServiceAccountName() + SecretSuffix
}

// RenderRBAC renders the RBAC configuration using a specified template.
func RenderRBAC(fsys fs.FS, cfg RBACConfig) ([]byte, error) {
	if cfg.PipelineName == "" || cfg.PipelineNamespace == "" {
		return nil, fmt.Errorf("invalid RBACConfig: PipelineName and PipelineNamespace must not be empty")
	}
	return renderTemplate(fsys, RBACTemplateFileName, RBACTemplateName, cfg)
}
