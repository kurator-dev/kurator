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
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	fakedisco "k8s.io/client-go/discovery/fake"
	coretesting "k8s.io/client-go/testing"
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

// can have nodes, pods  and configMap in a fake clientset unit testing
func newFakeDiscovery() discovery.DiscoveryInterface {
	fakeDiscoveryClient := &fakedisco.FakeDiscovery{Fake: &coretesting.Fake{}}
	fakeDiscoveryClient.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: corev1.SchemeGroupVersion.String(),
			APIResources: []metav1.APIResource{
				{Name: "serviceaccounts", Namespaced: true, Kind: "ServiceAccount"},
			},
		},
		{
			GroupVersion: corev1.SchemeGroupVersion.String(),
			APIResources: []metav1.APIResource{
				{Name: "configmaps", Namespaced: true, Kind: "ConfigMap"},
			},
		},
		{
			GroupVersion: corev1.SchemeGroupVersion.String(),
			APIResources: []metav1.APIResource{
				{Name: "services", Namespaced: true, Kind: "Service"},
			},
		},
		{
			GroupVersion: batchv1.SchemeGroupVersion.String(),
			APIResources: []metav1.APIResource{
				{Name: "jobs", Namespaced: true, Kind: "Job"},
			},
		},
		{
			GroupVersion: appsv1.SchemeGroupVersion.String(),
			APIResources: []metav1.APIResource{
				{Name: "deployments", Namespaced: true, Kind: "Deployment"},
			},
		},
	}

	return fakeDiscoveryClient
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

	kubeClinet := newFakeDiscovery()

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

	discoveryClient := newFakeDiscovery()

	err = AppendResourceSelector(discoveryClient, cpp, pp, resourceList)
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
