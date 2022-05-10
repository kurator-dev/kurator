package util

import (
	"bytes"
	"os"
	"path"
	"testing"

	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	"helm.sh/helm/v3/pkg/kube"
	"istio.io/istio/pkg/test/util/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestVolcano(t *testing.T) {
	b, err := os.ReadFile(path.Join("testdata", "volcano.yaml"))
	assert.NoError(t, err)

	helm := kube.New(nil)

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

	kubeClinet, err := helm.Factory.KubernetesClientSet()
	assert.NoError(t, err)

	err = AppendResourceSelector(kubeClinet, cpp, pp, resourceList)
	assert.NoError(t, err)

	expectedCPP := &policyv1alpha1.ClusterPropagationPolicy{}
	b, err = os.ReadFile(path.Join("testdata", "volcano-cpp.yaml"))
	assert.NoError(t, err)
	yaml.Unmarshal(b, expectedCPP)

	assert.Equal(t, expectedCPP, cpp)

	expectPP := &policyv1alpha1.PropagationPolicy{}
	b, err = os.ReadFile(path.Join("testdata", "volcano-pp.yaml"))
	assert.NoError(t, err)
	yaml.Unmarshal(b, expectPP)
	assert.Equal(t, expectPP, pp)
}

func TestIstioOperator(t *testing.T) {
	b, err := os.ReadFile(path.Join("testdata", "istio-operator.yaml"))
	assert.NoError(t, err)

	helm := kube.New(nil)

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

	kubeClinet, err := helm.Factory.KubernetesClientSet()
	assert.NoError(t, err)

	err = AppendResourceSelector(kubeClinet, cpp, pp, resourceList)
	assert.NoError(t, err)

	expectedCPP := &policyv1alpha1.ClusterPropagationPolicy{}
	b, err = os.ReadFile(path.Join("testdata", "istio-operator-cpp.yaml"))
	assert.NoError(t, err)
	yaml.Unmarshal(b, expectedCPP)

	assert.Equal(t, expectedCPP, cpp)

	expectPP := &policyv1alpha1.PropagationPolicy{}
	b, err = os.ReadFile(path.Join("testdata", "istio-operator-pp.yaml"))
	assert.NoError(t, err)
	yaml.Unmarshal(b, expectPP)
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
			obj: &v1.Secret{
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
