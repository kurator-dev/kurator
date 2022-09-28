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
	"bytes"
	"os"
	"path"
	"sync"
	"testing"

	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/kubectl/pkg/scheme"
)

var (
	nopLogger   = func(_ string, _ ...interface{}) {}
	addToScheme sync.Once
)

func newTestClient(t *testing.T) *kube.Client {
	addToScheme.Do(func() {
		if err := apiextv1.AddToScheme(scheme.Scheme); err != nil {
			// This should never happen.
			panic(err)
		}
		if err := apiextv1beta1.AddToScheme(scheme.Scheme); err != nil {
			panic(err)
		}
	})

	testFactory := cmdtesting.NewTestFactory()
	t.Cleanup(testFactory.Cleanup)

	c := &kube.Client{
		Factory: testFactory.WithNamespace("default"),
		Log:     nopLogger,
	}

	return c
}

func TestVolcano(t *testing.T) {
	b, err := os.ReadFile(path.Join("testdata", "volcano.yaml"))
	assert.NoError(t, err)

	helm := newTestClient(t)

	resourceList, err := helm.Build(bytes.NewBuffer(b), false)
	assert.NoError(t, err)

	clusters := []string{"fake-cluster"}
	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy.karmada.io/v1alpha1",
			Kind:       "ClusterPropagationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "volcano",
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: clusters,
				},
			},
		},
	}

	pp := &policyv1alpha1.PropagationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy.karmada.io/v1alpha1",
			Kind:       "PropagationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "volcano",
			Namespace: "volcano-system",
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: clusters,
				},
			},
		},
	}

	AppendResourceSelector(cpp, pp, resourceList)

	expectedCPP := &policyv1alpha1.ClusterPropagationPolicy{}
	b, err = os.ReadFile(path.Join("testdata", "volcano-cpp.yaml"))
	assert.NoError(t, err)
	err = yaml.Unmarshal(b, expectedCPP)
	assert.NoError(t, err)
	assert.Equal(t, expectedCPP, cpp)

	expectPP := &policyv1alpha1.PropagationPolicy{}
	b, err = os.ReadFile(path.Join("testdata", "volcano-pp.yaml"))
	assert.NoError(t, err)
	err = yaml.Unmarshal(b, expectPP)
	assert.NoError(t, err)
	assert.Equal(t, expectPP, pp)
}

func TestIstioOperator(t *testing.T) {
	b, err := os.ReadFile(path.Join("testdata", "istio-operator.yaml"))
	assert.NoError(t, err)

	helm := newTestClient(t)

	resourceList, err := helm.Build(bytes.NewBuffer(b), false)
	assert.NoError(t, err)

	clusters := []string{"fake-cluster"}
	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy.karmada.io/v1alpha1",
			Kind:       "ClusterPropagationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "istio-operator",
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: clusters,
				},
			},
		},
	}

	pp := &policyv1alpha1.PropagationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy.karmada.io/v1alpha1",
			Kind:       "PropagationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-operator",
			Namespace: "istio-operator",
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: clusters,
				},
			},
		},
	}

	AppendResourceSelector(cpp, pp, resourceList)

	expectedCPP := &policyv1alpha1.ClusterPropagationPolicy{}
	b, err = os.ReadFile(path.Join("testdata", "istio-operator-cpp.yaml"))
	assert.NoError(t, err)
	err = yaml.Unmarshal(b, expectedCPP)
	assert.NoError(t, err)

	assert.Equal(t, expectedCPP, cpp)

	expectPP := &policyv1alpha1.PropagationPolicy{}
	b, err = os.ReadFile(path.Join("testdata", "istio-operator-pp.yaml"))
	assert.NoError(t, err)
	err = yaml.Unmarshal(b, expectPP)
	assert.NoError(t, err)
	assert.Equal(t, expectPP, pp)
}

func TestGeneratePropagationPolicy(t *testing.T) {
	clusters := []string{"fake-cluster"}
	cases := []struct {
		name string
		obj  runtime.Object

		expected *policyv1alpha1.PropagationPolicy
	}{
		{
			name: "",
			obj: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-remote-secret-fake",
					Namespace: "istio-system",
				},
			},
			expected: &policyv1alpha1.PropagationPolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "policy.karmada.io/v1alpha1",
					Kind:       "PropagationPolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "istio-remote-secret-fake",
					Namespace: "istio-system",
				},
				Spec: policyv1alpha1.PropagationSpec{
					ResourceSelectors: []policyv1alpha1.ResourceSelector{
						{
							APIVersion: "v1",
							Kind:       "Secret",
							Name:       "istio-remote-secret-fake",
							Namespace:  "istio-system",
						},
					},
					Placement: policyv1alpha1.Placement{
						ClusterAffinity: &policyv1alpha1.ClusterAffinity{
							ClusterNames: clusters,
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := generatePropagationPolicy(clusters, tc.obj)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}
