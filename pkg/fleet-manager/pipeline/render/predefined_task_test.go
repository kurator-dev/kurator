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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pipelineapi "kurator.dev/kurator/pkg/apis/pipeline/v1alpha1"
)

func TestRenderPredefinedTask(t *testing.T) {
	expectedTaskFilePath := "testdata/predefined-task/"
	// Define test cases for various task templates and configurations.
	cases := []struct {
		name         string
		cfg          PredefinedTaskConfig
		expectError  bool
		expectedFile string
	}{
		// ---- Case: Default Configuration for Git Clone ----
		// This case tests the basic configuration of the 'git-clone' template.
		// It will not include auth, because auth will add in pipeline.
		{
			name: "git-clone with basic parameters",
			cfg: PredefinedTaskConfig{
				PipelineName: "test-pipeline",
				Namespace:    "kurator-pipeline",
				TemplateName: string(pipelineapi.GitClone),
				Params:       map[string]string{},
			},
			expectError:  false,
			expectedFile: "git-clone.yaml",
		},

		// ---- Case: Default Configuration for Go Test ----
		// This case tests the default configuration of the 'go-test' template.
		// It uses the default namespace and relies on all default parameter values.
		{
			name: "go-test with default parameters",
			cfg: PredefinedTaskConfig{
				PipelineName: "test-pipeline",
				Namespace:    "default",
				TemplateName: string(pipelineapi.GoTest),
				Params:       map[string]string{},
			},
			expectError:  false,
			expectedFile: "go-test-default.yaml",
		},

		// ---- Case: Custom Configuration for Go Test ----
		// This case customizes the 'go-test' template: setting Go version to 1.20,
		// targeting the './pkg/...' package path, and specifying the Linux ARM architecture.
		{
			name: "go-test with custom parameters - Go 1.20, ./pkg/..., Linux ARM",
			cfg: PredefinedTaskConfig{
				PipelineName: "test-pipeline",
				Namespace:    "kurator-pipeline",
				TemplateName: string(pipelineapi.GoTest),
				Params: map[string]string{
					"packages": "./pkg/...",
					"version":  "1.20",
					"GOOS":     "linux",
					"GOARCH":   "arm",
					"flags":    "-race -cover -v",
				},
			},
			expectError:  false,
			expectedFile: "go-test-custom-value.yaml",
		},

		// ---- Case: Custom Configuration for Go Lint ----
		// This case customizes the 'go-lint' template: setting golangci-lint version to latest,
		// just specifying additional linting flags.
		{
			name: "go-lint with custom parameters - latest version, ./src/..., extra flags",
			cfg: PredefinedTaskConfig{
				PipelineName: "test-pipeline",
				Namespace:    "kurator-pipeline",
				TemplateName: string(pipelineapi.GoLint),
				Params:       map[string]string{},
			},
			expectError:  false,
			expectedFile: "go-lint-custom-value.yaml",
		},

		// ---- Case: Advanced Custom Configuration for Go Lint ----
		// This case customizes the 'go-lint' template for a more complex scenario: setting
		// golangci-lint version to a specific older version (1.25.0), targeting a specific
		// package path './cmd/...', and specifying custom linting flags for that context.
		{
			name: "advanced go-lint custom configuration - version 1.25.0, ./cmd/..., specific flags",
			cfg: PredefinedTaskConfig{
				PipelineName: "test-pipeline",
				Namespace:    "kurator-pipeline",
				TemplateName: string(pipelineapi.GoLint),
				Params: map[string]string{
					"package": "./cmd/...",
					"version": "1.25.0",
					"flags":   "--enable=govet --enable=errcheck",
					"GOOS":    "linux",
					"GOARCH":  "arm",
				},
			},
			expectError:  false,
			expectedFile: "go-lint-advanced-config.yaml",
		},

		// ---- Case: default Configuration for build and upload image ----
		{
			name: "Custom Configuration for Kaniko Build Task",
			cfg: PredefinedTaskConfig{
				PipelineName: "test-pipeline",
				Namespace:    "kurator-pipeline",
				TemplateName: string(pipelineapi.BuildPushImage),
				Params: map[string]string{
					"image": "ghcr.io/test-orz/test-image:0.3.1",
				},
				OwnerReference: &metav1.OwnerReference{
					APIVersion: "v1",
					Kind:       "Deployment",
					Name:       "example-deployment",
					UID:        "22345678-1234-1234-1234-123456789abc",
				},
			},
			expectError:  false,
			expectedFile: "build-and-push-image-default.yaml",
		},

		{
			name: "Custom Dockerfile and Context for Kaniko Build Task",
			cfg: PredefinedTaskConfig{
				PipelineName: "test-pipeline",
				Namespace:    "kurator-pipeline",
				TemplateName: string(pipelineapi.BuildPushImage),
				Params: map[string]string{
					"image":      "ghcr.io/test-orz/test-image:0.3.2",
					"dockerfile": "./app/Dockerfile",
					"context":    "./app/",
				},
				OwnerReference: &metav1.OwnerReference{
					APIVersion: "v1",
					Kind:       "Deployment",
					Name:       "example-deployment",
					UID:        "22345678-1234-1234-1234-123456789abc",
				},
			},
			expectError:  false,
			expectedFile: "build-and-push-image-custom.yaml",
		},

		// TODO: Add more test cases here for different task templates or configurations
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := RenderPredefinedTask(tc.cfg)

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
