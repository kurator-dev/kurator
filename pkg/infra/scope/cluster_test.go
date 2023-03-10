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

package scope

import (
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	"kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
)

func TestStack(t *testing.T) {
	c := &Cluster{
		UID: "dcf78fcbb",
		NamespacedName: types.NamespacedName{
			Name:      "quickstart",
			Namespace: "default",
		},
	}

	gotSuffix := c.StackSuffix()
	gotStackName := c.StackName()

	g := NewWithT(t)
	g.Expect(gotSuffix).To(Equal(".dcf78fcbb.kurator.dev"))
	g.Expect(gotStackName).To(Equal("cf-dcf78fcbb-cluster-kurator-dev"))
}

func TestDeviceName(t *testing.T) {
	cases := []struct {
		idx      int
		infra    v1alpha1.ClusterInfraType
		expected string
	}{
		{
			idx:      0,
			infra:    "aws",
			expected: "/dev/sdb",
		},
		{
			idx:      1,
			infra:    "aws",
			expected: "/dev/sdc",
		},
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			actual := deviceName(tc.infra, tc.idx)
			g := NewWithT(t)

			g.Expect(tc.expected).To(Equal(actual))
		})
	}
}
