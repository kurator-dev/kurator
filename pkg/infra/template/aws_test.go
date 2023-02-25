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

	"kurator.dev/kurator/pkg/infra/scope"
)

func TestRenderClusterAPIForAWS(t *testing.T) {
	cases := []struct {
		name     string
		aws      *scope.Cluster
		expected string
	}{
		{
			name: "aws/capa-quickstart",
			aws: &scope.Cluster{
				UID:            "xxxxxx",
				InfraType:      "aws",
				NamespacedName: types.NamespacedName{Namespace: "default", Name: "capa-quickstart"},
				Version:        "v1.23.0",
				Region:         "us-east-1",
				VpcCIDR:        "10.10.0.0/16",
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
			name: "aws/without-sshkey",
			aws: &scope.Cluster{
				UID:            "xxxxxx",
				InfraType:      "aws",
				NamespacedName: types.NamespacedName{Namespace: "default", Name: "capa-quickstart"},
				Version:        "v1.23.0",
				Region:         "us-east-1",
				VpcCIDR:        "10.0.0.0/16",
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
		{
			name: "aws/enable-podidentity",
			aws: &scope.Cluster{
				UID:            "xxxxxx",
				InfraType:      "aws",
				NamespacedName: types.NamespacedName{Namespace: "default", Name: "capa-quickstart"},
				Version:        "v1.23.0",
				Region:         "us-east-1",
				VpcCIDR:        "10.0.0.0/16",
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
				EnablePodIdentity: true,
				BucketName:        "test-bucket",
			},
			expected: "enable-podidentity.yaml",
		},
		{
			name: "aws/with-volumes",
			aws: &scope.Cluster{
				UID:            "xxxxxx",
				InfraType:      "aws",
				NamespacedName: types.NamespacedName{Namespace: "default", Name: "capa-quickstart"},
				Version:        "v1.23.0",
				Region:         "us-east-1",
				VpcCIDR:        "10.0.0.0/16",
				PodCIDR:        []string{"192.168.0.0/16"},
				ServiceCIDR:    []string{"10.96.0.0/12"},
				ControlPlane: &scope.Instance{
					Replicas:     3,
					InstanceType: "t3.large",
					ImageOS:      "ubuntu-18.04",
					RootVolume: &scope.InstanceVolume{
						Size: 100,
						Type: "gp2",
					},
				},
				Workers: []*scope.Instance{
					{
						Replicas:     3,
						InstanceType: "t3.large",
						ImageOS:      "ubuntu-18.04",
						RootVolume: &scope.InstanceVolume{
							Size: 100,
							Type: "gp3",
						},
						DataVolumes: []scope.InstanceVolume{
							{
								DeviceName: "/dev/sdb1",
								Size:       200,
								Type:       "gp2",
							},
							{
								DeviceName: "/dev/sdb2",
								Size:       300,
								Type:       "gp3",
							},
						},
					},
				},
				EnablePodIdentity: true,
				BucketName:        "test-bucket",
			},
			expected: "with-volumes.yaml",
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
	data, err := os.ReadFile(fmt.Sprintf("testdata/%s", filename))
	if err != nil {
		panic(err)
	}
	return data
}
