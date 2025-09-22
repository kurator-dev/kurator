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
	"k8s.io/apimachinery/pkg/types"
)

func TestRenderTestloaderConfig(t *testing.T) {
	expectedTestDataFile := "testdata/"
	namespacedName := types.NamespacedName{
		Namespace: "test",
		Name:      "testloader",
	}
	annotationKey := "kurator.dev/rollout"
	annotationValue := "policy"
	cases := []struct {
		name              string
		constTemplateName string
		expectFileName    string
	}{
		{
			name:              "testloader deploy template",
			constTemplateName: TestlaoderDeployment,
			expectFileName:    "testloader-deploy.yaml",
		},
		{
			name:              "testloader svc template",
			constTemplateName: TestlaoderService,
			expectFileName:    "testloader-svc.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := RenderTestloaderConfig(tc.constTemplateName, namespacedName, annotationKey, annotationValue)
			if err != nil {
				assert.Error(t, err)
			} else {
				expected, err := os.ReadFile(expectedTestDataFile + tc.expectFileName)
				assert.NoError(t, err)
				assert.Equal(t, result, expected)
			}
		})
	}
}
