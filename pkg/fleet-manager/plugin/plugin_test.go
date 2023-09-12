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
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

func TestRenderKyvernoPolicy(t *testing.T) {
	cases := []struct {
		name  string
		fleet types.NamespacedName
		ref   *metav1.OwnerReference
		in    *v1alpha1.KyvernoConfig
	}{
		{
			name: "default",
			fleet: types.NamespacedName{
				Name:      "fleet-1",
				Namespace: "default",
			},
			ref: &metav1.OwnerReference{
				APIVersion: v1alpha1.GroupVersion.String(),
				Kind:       "Fleet",
				Name:       "fleet-1",
				UID:        "xxxxxx",
			},
			in: &v1alpha1.KyvernoConfig{
				PodSecurity: &v1alpha1.PodSecurityPolicy{
					Standard:                "baseline",
					Severity:                "medium",
					ValidationFailureAction: "Audit",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderKyvernoPolicy(manifestFS, tc.fleet, tc.ref, FleetCluster{
				Name:       "cluster1",
				SecretName: "cluster1",
				SecretKey:  "kubeconfig.yaml",
			}, tc.in)
			assert.NoError(t, err)

			getExpected, err := getExpected(KyvernoPolicyComponentName, tc.name)
			assert.NoError(t, err)

			assert.Equal(t, string(getExpected), string(got))
		})
	}
}

func TestRenderKyverno(t *testing.T) {
	cases := []struct {
		name  string
		fleet types.NamespacedName
		ref   *metav1.OwnerReference
		in    *v1alpha1.KyvernoConfig
	}{
		{
			name: "default",
			fleet: types.NamespacedName{
				Name:      "fleet-1",
				Namespace: "default",
			},
			ref: &metav1.OwnerReference{
				APIVersion: v1alpha1.GroupVersion.String(),
				Kind:       "Fleet",
				Name:       "fleet-1",
				UID:        "xxxxxx",
			},
			in: &v1alpha1.KyvernoConfig{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderKyverno(manifestFS, tc.fleet, tc.ref, FleetCluster{
				Name:       "cluster1",
				SecretName: "cluster1",
				SecretKey:  "kubeconfig.yaml",
			}, tc.in)
			assert.NoError(t, err)

			getExpected, err := getExpected("kyverno", tc.name)
			assert.NoError(t, err)

			assert.Equal(t, string(getExpected), string(got))
		})
	}
}

func TestRenderGrafana(t *testing.T) {
	cases := []struct {
		name    string
		fleet   types.NamespacedName
		ref     *metav1.OwnerReference
		cfg     *v1alpha1.GrafanaConfig
		sources []*GrafanaDataSource
	}{
		{
			name: "default",
			fleet: types.NamespacedName{
				Name:      "fleet-1",
				Namespace: "default",
			},
			ref: &metav1.OwnerReference{
				APIVersion: v1alpha1.GroupVersion.String(),
				Kind:       "Fleet",
				Name:       "fleet-1",
				UID:        "xxxxxx",
			},
			cfg: &v1alpha1.GrafanaConfig{},
		},
		{
			name: "with-datasource",
			fleet: types.NamespacedName{
				Name:      "fleet-1",
				Namespace: "default",
			},
			ref: &metav1.OwnerReference{
				APIVersion: v1alpha1.GroupVersion.String(),
				Kind:       "Fleet",
				Name:       "fleet-1",
				UID:        "xxxxxx",
			},
			cfg: &v1alpha1.GrafanaConfig{},
			sources: []*GrafanaDataSource{
				{
					Name:       "prometheus",
					SourceType: "prometheus",
					URL:        "http://prometheus:9090",
					Access:     "proxy",
					IsDefault:  true,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderGrafana(manifestFS, tc.fleet, tc.ref, tc.cfg, tc.sources)
			assert.NoError(t, err)

			getExpected, err := getExpected("grafana", tc.name)
			assert.NoError(t, err)

			assert.Equal(t, string(getExpected), string(got))
		})
	}
}

func TestRenderThanos(t *testing.T) {
	cases := []struct {
		name  string
		fleet types.NamespacedName
		ref   *metav1.OwnerReference
		in    *v1alpha1.MetricConfig
	}{
		{
			name: "default",
			fleet: types.NamespacedName{
				Name:      "fleet-1",
				Namespace: "default",
			},
			in: &v1alpha1.MetricConfig{
				Thanos: v1alpha1.ThanosConfig{
					ObjectStoreConfig: v1alpha1.ObjectStoreConfig{
						SecretName: "thanos-objstore",
					},
				},
			},
		},
		{
			name: "custom-values",
			fleet: types.NamespacedName{
				Name:      "fleet-1",
				Namespace: "monitoring",
			},
			in: &v1alpha1.MetricConfig{
				Thanos: v1alpha1.ThanosConfig{
					Chart: &v1alpha1.ChartConfig{
						Repository: "https://charts.bitnami.com/bitnami",
						Name:       "thanos",
						Version:    "x.y.z",
					},
					ObjectStoreConfig: v1alpha1.ObjectStoreConfig{
						SecretName: "thanos-objstore",
					},
					ExtraArgs: apiextensionsv1.JSON{
						Raw: []byte("{\"key1\":\"value1\"}"),
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderThanos(manifestFS, tc.fleet, tc.ref, tc.in)
			assert.NoError(t, err)

			getExpected, err := getExpected("thanos", tc.name)
			assert.NoError(t, err)

			assert.Equal(t, string(getExpected), string(got))
		})
	}
}

func TestRenderPrometheus(t *testing.T) {
	cases := []struct {
		name  string
		fleet types.NamespacedName
		ref   *metav1.OwnerReference
		in    *v1alpha1.MetricConfig
	}{
		{
			name: "default",
			fleet: types.NamespacedName{
				Name:      "fleet-1",
				Namespace: "default",
			},
			ref: &metav1.OwnerReference{
				APIVersion: v1alpha1.GroupVersion.String(),
				Kind:       "Fleet",
				Name:       "fleet-1",
				UID:        "xxxxxx",
			},
			in: &v1alpha1.MetricConfig{},
		},
		{
			name: "with-values",
			fleet: types.NamespacedName{
				Name:      "fleet-1",
				Namespace: "default",
			},
			in: &v1alpha1.MetricConfig{
				Prometheus: v1alpha1.PrometheusConfig{
					ExtraArgs: apiextensionsv1.JSON{
						Raw: []byte("{\"key1\":\"value1\"}"),
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderPrometheus(manifestFS, tc.fleet, tc.ref, FleetCluster{
				Name:       "cluster1",
				SecretName: "cluster1",
				SecretKey:  "kubeconfig.yaml",
			}, tc.in)
			assert.NoError(t, err)

			getExpected, err := getExpected("prometheus", tc.name)
			assert.NoError(t, err)

			assert.Equal(t, string(getExpected), string(got))
		})
	}
}

func TestRenderVelero(t *testing.T) {
	cases := []struct {
		name  string
		fleet types.NamespacedName
		ref   *metav1.OwnerReference
		in    *v1alpha1.BackupConfig
	}{
		{
			name: "default",
			fleet: types.NamespacedName{
				Name:      "fleet-1",
				Namespace: "default",
			},
			ref: &metav1.OwnerReference{
				APIVersion: v1alpha1.GroupVersion.String(),
				Kind:       "Fleet",
				Name:       "fleet-1",
				UID:        "xxxxxx",
			},
			in: &v1alpha1.BackupConfig{
				Storage: v1alpha1.BackupStorage{
					Location: v1alpha1.BackupStorageLocation{
						Bucket:   "velero",
						Provider: "aws",
						Endpoint: "http://x.x.x.x:x",
						Region:   "minio",
					},
					SecretName: "backup-secret",
				},
			},
		},
		{
			name: "custom-values",
			fleet: types.NamespacedName{
				Name:      "fleet-1",
				Namespace: "default",
			},
			ref: &metav1.OwnerReference{
				APIVersion: v1alpha1.GroupVersion.String(),
				Kind:       "Fleet",
				Name:       "fleet-1",
				UID:        "xxxxxx",
			},
			in: &v1alpha1.BackupConfig{
				Storage: v1alpha1.BackupStorage{
					Location: v1alpha1.BackupStorageLocation{
						Bucket:   "velero",
						Provider: "aws",
						Endpoint: "http://x.x.x.x:x",
						Region:   "minio",
					},
					SecretName: "backup-secret",
				},
				ExtraArgs: apiextensionsv1.JSON{
					Raw: []byte("{\"image\": {\n  \"repository\": \"velero/velero\",\n  \"tag\": \"v1.10.1\",\n  \"pullPolicy\": \"IfNotPresent\"\n}}"),
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderVelero(manifestFS, tc.fleet, tc.ref, FleetCluster{
				Name:       "cluster1",
				SecretName: "cluster1",
				SecretKey:  "kubeconfig.yaml",
			}, tc.in, "xxx")
			assert.NoError(t, err)

			getExpected, err := getExpected("backup", tc.name)
			assert.NoError(t, err)

			assert.Equal(t, string(getExpected), string(got))
		})
	}
}
