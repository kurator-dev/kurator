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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestRenderCustomTask(t *testing.T) {
	expectedTaskFilePath := "testdata/custom-task/"
	// Define test cases for various task templates and configurations.
	cases := []struct {
		name         string
		cfg          CustomTaskConfig
		expectError  bool
		expectedFile string
	}{
		// ---- Case: Default Configuration for Go Test ----
		// This case tests a simple configuration of to print the repo readme.
		{
			name: "cat-readme",
			cfg: CustomTaskConfig{
				TaskName:          "cat-readme",
				PipelineName:      "test-pipeline",
				PipelineNamespace: "default",
				Image:             "zshusers/zsh:4.3.15",
				Command: []string{
					"/bin/sh",
					"-c",
				},
				Args: []string{
					"cat $(workspaces.source.path)/README.md",
				},
			},
			expectError:  false,
			expectedFile: "cat-readme.yaml",
		},
		{
			name: "minimal-configuration",
			cfg: CustomTaskConfig{
				TaskName:          "minimal-task",
				PipelineName:      "test-pipeline",
				PipelineNamespace: "default",
				Image:             "alpine:latest",
			},
			expectError:  false,
			expectedFile: "minimal-task.yaml",
		},
		{
			name: "complete-configuration",
			cfg: CustomTaskConfig{
				TaskName:          "complete-task",
				PipelineName:      "test-pipeline",
				PipelineNamespace: "default",
				Image:             "python:3.8",
				Command:           []string{"python", "-c"},
				Args:              []string{"print('Hello World')"},
				Env:               []corev1.EnvVar{{Name: "ENV_VAR", Value: "test"}},
				ResourceRequirements: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
				},
				Script: "print('This is a complete test')",
			},
			expectError:  false,
			expectedFile: "complete-task.yaml",
		},
		{
			name: "missing-required-fields-test-pipeline",
			cfg: CustomTaskConfig{
				PipelineNamespace: "default",
			},
			expectError:  true,
			expectedFile: "",
		},
		{
			name: "with-environment-variables-test-pipeline",
			cfg: CustomTaskConfig{
				TaskName:          "env-task",
				PipelineName:      "test-pipeline",
				PipelineNamespace: "default",
				Image:             "node:14",
				Env:               []corev1.EnvVar{{Name: "NODE_ENV", Value: "production"}},
			},
			expectError:  false,
			expectedFile: "env-task.yaml",
		},
		{
			name: "with-resource-requirements-test-pipeline",
			cfg: CustomTaskConfig{
				TaskName:          "resource-task",
				PipelineName:      "test-pipeline",
				PipelineNamespace: "default",
				Image:             "golang:1.16",
				ResourceRequirements: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
			},
			expectError:  false,
			expectedFile: "resource-task.yaml",
		},
		{
			name: "with-commands-and-arguments-test-pipeline",
			cfg: CustomTaskConfig{
				TaskName:          "cmd-args",
				PipelineName:      "test-pipeline",
				PipelineNamespace: "default",
				Image:             "ubuntu:latest",
				Command:           []string{"/bin/bash", "-c"},
				Args:              []string{"echo 'Hello from command'"},
			},
			expectError:  false,
			expectedFile: "cmd-args-task.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := RenderCustomTask(tc.cfg)

			// Test assertions
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				expected, err := os.ReadFile(expectedTaskFilePath + tc.expectedFile)
				assert.NoError(t, err)
				assert.Equal(t, string(expected), string(result))
			}
		})
	}
}
