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

	pipelineapi "kurator.dev/kurator/pkg/apis/pipeline/v1alpha1"
)

const (
	// GitCloneTask is the predefined task template name of git clone task
	GitCloneTask = "git-clone"
	// GoTestTask is the predefined task template name of go test task
	GoTestTask = "go-test"
	// GoLintTask is the predefined task template name of golangci lint task
	GoLintTask = "go-lint"
	// BuildPushImage is the predefined task template name of task about building image and pushing it to image repo
	BuildPushImage = "build-and-push-image"
)

type PredefinedTaskConfig struct {
	PipelineName      string
	PipelineNamespace string
	// TemplateName is set by user in `Pipeline.Tasks[i].PredefinedTask.Name`
	TemplateName string
	// Params is set by user in `Pipeline.Tasks[i].PredefinedTask.Params`
	Params         map[string]string
	OwnerReference *metav1.OwnerReference
}

// PredefinedTaskName is the name of Predefined task object, in case different pipeline have the same name task.
func (cfg PredefinedTaskConfig) PredefinedTaskName() string {
	return cfg.TemplateName + "-" + cfg.PipelineName
}

// RenderPredefinedTaskWithPipeline takes a Pipeline object and generates YAML byte array configuration representing the PredefinedTask configuration.
func RenderPredefinedTaskWithPipeline(pipeline *pipelineapi.Pipeline, task *pipelineapi.PredefinedTask) ([]byte, error) {
	cfg := PredefinedTaskConfig{
		PipelineName:      pipeline.Name,
		PipelineNamespace: pipeline.Namespace,
		TemplateName:      string(task.Name),
		Params:            task.Params,
		OwnerReference:    GeneratePipelineOwnerRef(pipeline),
	}

	return RenderPredefinedTask(cfg)
}

// RenderPredefinedTask takes a PredefinedTaskConfig object and generates YAML byte array configuration representing the PredefinedTask configuration.
func RenderPredefinedTask(cfg PredefinedTaskConfig) ([]byte, error) {
	templateContent, ok := predefinedTaskTemplates[cfg.TemplateName]
	if !ok {
		return nil, fmt.Errorf("predefinedTask template content named '%s' not found", cfg.TemplateName)
	}

	return renderTemplate(templateContent, generateTaskTemplateName(cfg.TemplateName), cfg)
}

func generateTaskTemplateName(taskType string) string {
	return "pipeline " + taskType + " task template"
}

var predefinedTaskTemplates = map[string]string{
	GitCloneTask:   GitCloneTaskContent,
	GoTestTask:     GoTestTaskContent,
	GoLintTask:     GoLintTaskContent,
	BuildPushImage: BuildPushImageContent,
}
