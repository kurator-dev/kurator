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

package template

import (
	"fmt"
	"os"
	"testing"

	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"kurator.dev/kurator/pkg/controllers/scope"
)

func TestRenderCNI(t *testing.T) {
	cases := []struct {
		name     string
		cni      scope.CNI
		expected string
	}{
		{
			name: "calico",
			cni: scope.CNI{
				Namespace: "default",
				Name:      "calico",
				Type:      "calico",
			},
			expected: "calico.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := RenderCNI(tc.cni)

			g := gomega.NewWithT(t)
			g.Expect(err).To(gomega.BeNil())

			assert.Equal(t, string(readCNITestData(tc.expected)), string(actual))
		})
	}
}

func readCNITestData(filename string) []byte {
	data, err := os.ReadFile(fmt.Sprintf("testdata/cni/%s", filename))
	if err != nil {
		panic(err)
	}
	return data
}
