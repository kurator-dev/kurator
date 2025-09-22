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

func TestRenderPipelineWithPipeline(t *testing.T) {
	expectedPipelineFilePath := "testdata/pipeline/"
	testPipeline := pipelineapi.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pipeline",
			Namespace: "kurator-pipeline",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Deployment",
					Name:       "example-deployment",
					UID:        "22345678-1234-1234-1234-123456789abc",
				},
			},
		},
	}
	cases := []struct {
		name         string
		tasks        []pipelineapi.PipelineTask
		expectError  bool
		expectedFile string
	}{
		{
			name: "valid pipeline configuration, contains tasks: git-clone, cat-readme, go-test",
			tasks: []pipelineapi.PipelineTask{
				{
					Name:           "git-clone",
					PredefinedTask: &pipelineapi.PredefinedTask{Name: pipelineapi.GitClone},
				},
				{
					Name:       "cat-readme",
					CustomTask: &pipelineapi.CustomTask{},
				},
				{
					Name:           "go-test",
					PredefinedTask: &pipelineapi.PredefinedTask{Name: pipelineapi.GoTest},
				},
			},
			expectError:  false,
			expectedFile: "readme-test.yaml",
		},
		{
			name: "valid comprehensive pipeline configuration, contains tasks: git-clone, cat-readme, go-test, go-lint, push-and-build-image",
			tasks: []pipelineapi.PipelineTask{
				{
					Name:           "git-clone",
					PredefinedTask: &pipelineapi.PredefinedTask{Name: pipelineapi.GitClone},
				},
				{
					Name:       "cat-readme",
					CustomTask: &pipelineapi.CustomTask{},
				},
				{
					Name:           "go-test",
					PredefinedTask: &pipelineapi.PredefinedTask{Name: pipelineapi.GoTest},
				},
				{
					Name:           "go-lint",
					PredefinedTask: &pipelineapi.PredefinedTask{Name: pipelineapi.GoLint},
				},
				{
					Name:           "build-and-push-image",
					PredefinedTask: &pipelineapi.PredefinedTask{Name: pipelineapi.BuildPushImage},
				},
			},
			expectError:  false,
			expectedFile: "comprehensive.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testPipeline.Spec.Tasks = tc.tasks

			result, err := RenderPipelineWithPipeline(&testPipeline)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				expected, err := os.ReadFile(expectedPipelineFilePath + tc.expectedFile)
				assert.NoError(t, err)
				assert.Equal(t, string(expected), string(result))
			}
		})
	}
}
