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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"kurator.dev/kurator/pkg/fleet-manager/manifests"
)

var manifestFS = manifests.BuiltinOrDir("manifests")

const expectedRBACFilePath = "testdata/rbac/"

func TestRenderRBAC(t *testing.T) {
	// Define test cases including both valid and error scenarios.
	cases := []struct {
		name         string
		cfg          RBACConfig
		expectError  bool
		expectedFile string
	}{
		{
			name: "valid configuration",
			cfg: RBACConfig{
				PipelineName:      "example",
				PipelineNamespace: "default",
			},
			expectError:  false,
			expectedFile: "default-example.yaml",
		},
		{
			name: "empty PipelineName",
			cfg: RBACConfig{
				PipelineName:      "",
				PipelineNamespace: "default",
			},
			expectError: true,
		},
		{
			name: "empty PipelineNamespace",
			cfg: RBACConfig{
				PipelineName:      "example",
				PipelineNamespace: "",
			},
			expectError: true,
		},
		{
			name: "invalid file system path",
			cfg: RBACConfig{
				PipelineName:      "example",
				PipelineNamespace: "default",
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := manifestFS
			// Use an invalid file system for the relevant test case.
			if tc.name == "invalid file system path" {
				fs = manifests.BuiltinOrDir("invalid-path")
			}

			result, err := renderRBAC(fs, tc.cfg)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				expected, err := os.ReadFile(expectedRBACFilePath + tc.expectedFile)
				assert.NoError(t, err)
				assert.Equal(t, string(expected), string(result))
			}
		})
	}
}
