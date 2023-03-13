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

package plugin

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	"kurator.dev/kurator/pkg/infra/scope"
)

func TestRenderCNI(t *testing.T) {
	c, err := readClusterFromYaml("cni-extra.yaml")
	assert.NoError(t, err)

	cases := []struct {
		name     string
		cluster  *scope.Cluster
		expected string
	}{
		{
			name: "aws-cni-calico",
			cluster: &scope.Cluster{
				InfraType: "aws",
				UID:       "xxxxx",
				NamespacedName: types.NamespacedName{
					Name:      "test",
					Namespace: "default",
				},
				CNIType: "calico",
				Cluster: c,
			},
			expected: "aws-cni-calico.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := RenderCNI(tc.cluster)
			assert.NoError(t, err)
			assert.Equal(t, string(readCNITestData(tc.expected)), string(actual))
		})
	}
}

func TestRenderCNIWithCustomTemplate(t *testing.T) {
	c, err := readClusterFromYaml("cni-extra.yaml")
	assert.NoError(t, err)

	pluginYAMLFunc = readCNITemplate
	cases := []struct {
		name     string
		cluster  *scope.Cluster
		expected string
	}{
		{
			name: "aws-cni-custom",
			cluster: &scope.Cluster{
				InfraType: "aws",
				UID:       "xxxxx",
				NamespacedName: types.NamespacedName{
					Name:      "test",
					Namespace: "default",
				},
				CNIType: "custom",
				Cluster: c,
			},
			expected: "aws-cni-custom.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := RenderCNI(tc.cluster)
			assert.NoError(t, err)
			assert.Equal(t, string(readCNITestData(tc.expected)), string(actual))
		})
	}
}

func readCNITemplate(_ string) (string, error) {
	data, err := os.ReadFile(path.Join("testdata", "aws-cni-custom.yaml.tpl"))
	if err != nil {
		panic(err)
	}
	return string(data), nil
}

func readCNITestData(filename string) []byte {
	data, err := os.ReadFile(fmt.Sprintf("testdata/%s", filename))
	if err != nil {
		panic(err)
	}
	return data
}

func readClusterFromYaml(name string) (*v1alpha1.Cluster, error) {
	filename := path.Join("testdata", "cluster", name)
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := &v1alpha1.Cluster{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}

	return c, nil
}
