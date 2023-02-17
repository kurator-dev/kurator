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
