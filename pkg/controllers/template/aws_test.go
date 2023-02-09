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
	"k8s.io/apimachinery/pkg/types"

	"kurator.dev/kurator/pkg/controllers/scope"
)

func TestRenderClusterAPIForAWS(t *testing.T) {
	cases := []struct {
		name     string
		aws      *scope.Cluster
		expected string
	}{
		{
			name: "capa-quickstart",
			aws: &scope.Cluster{
				NamespacedName: types.NamespacedName{Namespace: "default", Name: "capa-quickstart"},
				Version:        "v1.23.0",
				Region:         "us-east-1",
				Credential:     "capa-quickstart-xxxxx",
				PodCIDR:        []string{"192.168.0.0/16"},
				ServiceCIDR:    []string{"10.96.0.0/12"},
				ControlPlane: &scope.Instance{
					Replicas:     1,
					InstanceType: "t3.large",
					SSHKey:       "default",
					ImageOS:      "ubuntu-18.04",
				},
				Workers: []*scope.Instance{
					{
						Replicas:     2,
						InstanceType: "t3.large",
						SSHKey:       "default",
						ImageOS:      "ubuntu-18.04",
					},
				},
			},
			expected: "capa-quickstart.yaml",
		},
		{
			name: "without-sshkey",
			aws: &scope.Cluster{
				NamespacedName: types.NamespacedName{Namespace: "default", Name: "capa-quickstart"},
				Version:        "v1.23.0",
				Region:         "us-east-1",
				Credential:     "capa-quickstart-xxxxx",
				PodCIDR:        []string{"192.168.0.0/16"},
				ServiceCIDR:    []string{"10.96.0.0/12"},
				ControlPlane: &scope.Instance{
					Replicas:     3,
					InstanceType: "t3.large",
					ImageOS:      "ubuntu-18.04",
				},
				Workers: []*scope.Instance{
					{
						Replicas:     3,
						InstanceType: "t3.large",
						ImageOS:      "ubuntu-18.04",
					},
				},
			},
			expected: "without-sshkey.yaml",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := RenderClusterAPIForAWS(tc.aws)

			g := gomega.NewWithT(t)
			g.Expect(err).To(gomega.BeNil())

			assert.Equal(t, string(readClusterAPITestData(tc.expected)), string(actual))
		})
	}
}

func readClusterAPITestData(filename string) []byte {
	data, err := os.ReadFile(fmt.Sprintf("testdata/clusterapi/%s", filename))
	if err != nil {
		panic(err)
	}
	return data
}
