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

package util

import (
	"testing"

	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"

	infrav1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
)

func TestGenerateUID(t *testing.T) {
	cases := []struct {
		left, right types.NamespacedName
		exceptEqual bool
	}{
		{types.NamespacedName{Namespace: "a", Name: "b-c"}, types.NamespacedName{Namespace: "a", Name: "b-c"}, true},
		{types.NamespacedName{Namespace: "a", Name: "b-d"}, types.NamespacedName{Namespace: "a", Name: "b-c"}, false},
		{types.NamespacedName{Namespace: "a", Name: "b-c"}, types.NamespacedName{Namespace: "a-b", Name: "c"}, false},
		{types.NamespacedName{Namespace: "a-b-c", Name: "d"}, types.NamespacedName{Namespace: "a-b", Name: "c-d"}, false},
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			gotLeft := GenerateUID(tc.left)
			gotRight := GenerateUID(tc.right)

			g := gomega.NewWithT(t)
			if tc.exceptEqual {
				g.Expect(gotLeft).To(gomega.Equal(gotRight))
			} else {
				g.Expect(gotLeft).ToNot(gomega.Equal(gotRight))
			}
		})
	}
}

func TestAdditionalResources(t *testing.T) {
	cases := []struct {
		input    *infrav1.Cluster
		expected []addonsv1.ResourceRef
	}{
		{
			input: &infrav1.Cluster{
				Spec: infrav1.ClusterSpec{
					AdditionalResources: []infrav1.ResourceRef{
						{
							Kind: "Secret",
							Name: "kurator-secret1",
						},
						{
							Kind: "ConfigMap",
							Name: "kurator-config1",
						},
						{
							Kind: "Secret",
							Name: "kurator-secret2",
						},
						{
							Kind: "ConfigMap",
							Name: "kurator-config2",
						},
					},
				},
			},
			expected: []addonsv1.ResourceRef{
				{
					Kind: "Secret",
					Name: "kurator-secret1",
				},
				{
					Kind: "ConfigMap",
					Name: "kurator-config1",
				},
				{
					Kind: "Secret",
					Name: "kurator-secret2",
				},
				{
					Kind: "ConfigMap",
					Name: "kurator-config2",
				},
			},
		},
		{
			input: &infrav1.Cluster{
				Spec: infrav1.ClusterSpec{
					AdditionalResources: []infrav1.ResourceRef{
						{
							Kind: "ConfigMap",
							Name: "kurator-config2",
						},
						{
							Kind: "Secret",
							Name: "kurator-secret1",
						},
						{
							Kind: "ConfigMap",
							Name: "kurator-config1",
						},
						{
							Kind: "Secret",
							Name: "kurator-secret2",
						},
					},
				},
			},
			expected: []addonsv1.ResourceRef{
				{
					Kind: "ConfigMap",
					Name: "kurator-config2",
				},
				{
					Kind: "Secret",
					Name: "kurator-secret1",
				},
				{
					Kind: "ConfigMap",
					Name: "kurator-config1",
				},
				{
					Kind: "Secret",
					Name: "kurator-secret2",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			got := AdditionalResources(tc.input)

			g := gomega.NewWithT(t)
			g.Expect(got).To(gomega.Equal(tc.expected))
		})
	}
}
