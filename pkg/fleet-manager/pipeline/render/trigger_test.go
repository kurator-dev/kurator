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
)

func TestRenderTrigger(t *testing.T) {
	expectedRBACFilePath := "testdata/trigger/"
	cases := []struct {
		name         string
		cfg          TriggerConfig
		expectError  bool
		expectedFile string
	}{
		{
			name: "default trigger configuration",
			cfg: TriggerConfig{
				PipelineName:      "test-pipeline",
				PipelineNamespace: "kurator-pipeline",
			},
			expectError:  false,
			expectedFile: "default.yaml",
		},
		{
			name: "trigger configuration with custom volume claim",
			cfg: TriggerConfig{
				PipelineName:      "test-pipeline",
				PipelineNamespace: "kurator-pipeline",
				AccessMode:        "ReadWriteOnce",
				StorageRequest:    "500Mi",
				StorageClassName:  "manual",
				VolumeMode:        "Filesystem",
			},
			expectError:  false,
			expectedFile: "with-volume-claim.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := RenderTrigger(tc.cfg)

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
