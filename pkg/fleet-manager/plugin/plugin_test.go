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
	corev1 "k8s.io/api/core/v1"
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
			got, err := RenderKyvernoPolicy(manifestFS, tc.fleet, tc.ref, KubeConfigSecretRef{
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
			got, err := RenderKyverno(manifestFS, tc.fleet, tc.ref, KubeConfigSecretRef{
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
			got, err := RenderPrometheus(manifestFS, tc.fleet, tc.ref, KubeConfigSecretRef{
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
		name          string
		fleet         types.NamespacedName
		ref           *metav1.OwnerReference
		in            *v1alpha1.BackupConfig
		newSecretName string
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
			newSecretName: "kurator-velero-s3",
		},
		{
			name: "custom-values-s3",
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
						Config: map[string]string{
							"s3Url":  "s3.us-east-2.amazonaws.com",
							"region": "us-east-2",
						},
					},
					SecretName: "backup-secret",
				},
				ExtraArgs: apiextensionsv1.JSON{
					Raw: []byte("{\"image\": {\n  \"repository\": \"velero/velero\",\n  \"tag\": \"v1.10.1\",\n  \"pullPolicy\": \"IfNotPresent\"\n}}"),
				},
			},
			newSecretName: "kurator-velero-s3",
		},
		{
			name: "custom-values-obs",
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
						Bucket:   "kurator-backup",
						Provider: "huaweicloud",
						Endpoint: "http://obs.cn-south-1.myhuaweicloud.com",
						Region:   "cn-south-1",
					},
					SecretName: "backup-secret",
				},
				ExtraArgs: apiextensionsv1.JSON{
					Raw: []byte("{\"image\": {\n  \"repository\": \"velero/velero\",\n  \"tag\": \"v1.10.1\",\n  \"pullPolicy\": \"IfNotPresent\"\n}}"),
				},
			},
			newSecretName: "kurator-velero-obs",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderVelero(manifestFS, tc.fleet, tc.ref, KubeConfigSecretRef{
				Name:       "cluster1",
				SecretName: "cluster1",
				SecretKey:  "kubeconfig.yaml",
			}, tc.in, tc.newSecretName)
			assert.NoError(t, err)

			getExpected, err := getExpected("backup", tc.name)
			assert.NoError(t, err)

			assert.Equal(t, string(getExpected), string(got))
		})
	}
}

func TestRenderStorageOperator(t *testing.T) {
	configPath := "/var/lib/rook"
	configPathPoint := &configPath
	monitorCount := 3
	managerCount := 2

	cases := []struct {
		name   string
		fleet  types.NamespacedName
		ref    *metav1.OwnerReference
		config *v1alpha1.DistributedStorageConfig
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
			config: &v1alpha1.DistributedStorageConfig{
				Storage: &v1alpha1.DistributedStorage{
					DataDirHostPath: configPathPoint,
					Monitor: &v1alpha1.MonSpec{
						Count: &monitorCount,
					},
					Manager: &v1alpha1.MgrSpec{
						Count: &managerCount,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderStorageOperator(manifestFS, tc.fleet, tc.ref, KubeConfigSecretRef{
				Name:       "cluster1",
				SecretName: "cluster1",
				SecretKey:  "kubeconfig.yaml",
			}, tc.config)
			assert.NoError(t, err)

			getExpected, err := getExpected("distributedstorage", tc.name)
			assert.NoError(t, err)

			assert.Equal(t, string(getExpected), string(got))
		})
	}
}

func TestRenderClusterStorage(t *testing.T) {
	configPath := "/var/lib/rook"
	configPathPoint := &configPath
	monitorCount := 3
	managerCount := 2

	cases := []struct {
		name   string
		fleet  types.NamespacedName
		ref    *metav1.OwnerReference
		config *v1alpha1.DistributedStorageConfig
	}{
		{
			name: "ceph-cluster-default",
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
			config: &v1alpha1.DistributedStorageConfig{
				Storage: &v1alpha1.DistributedStorage{
					DataDirHostPath: configPathPoint,
					Monitor: &v1alpha1.MonSpec{
						Count: &monitorCount,
					},
					Manager: &v1alpha1.MgrSpec{
						Count: &managerCount,
					},
				},
			},
		}, {
			name: "ceph-cluster-handle",
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
			config: &v1alpha1.DistributedStorageConfig{
				Storage: &v1alpha1.DistributedStorage{
					DataDirHostPath: configPathPoint,
					Monitor: &v1alpha1.MonSpec{
						Count: &monitorCount,
						Annotations: map[string]string{
							"role": "MonitorNodeAnnotation",
						},
						Labels: map[string]string{
							"role": "MonitorNodeLabel",
						},
						Placement: &v1alpha1.Placement{
							Tolerations: []corev1.Toleration{
								{
									Key:      "storage-node",
									Operator: corev1.TolerationOpExists,
								},
							},
						},
					},
					Manager: &v1alpha1.MgrSpec{
						Count: &managerCount,
						Annotations: map[string]string{
							"role": "ManagerNodeAnnotation",
						},
						Labels: map[string]string{
							"role": "ManagerNodeLabel",
						},
						Placement: &v1alpha1.Placement{
							Tolerations: []corev1.Toleration{
								{
									Key:      "storage-node",
									Operator: corev1.TolerationOpExists,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderClusterStorage(manifestFS, tc.fleet, tc.ref, KubeConfigSecretRef{
				Name:       "cluster1",
				SecretName: "cluster1",
				SecretKey:  "kubeconfig.yaml",
			}, tc.config)
			assert.NoError(t, err)

			getExpected, err := getExpected("distributedstorage", tc.name)
			assert.NoError(t, err)

			assert.Equal(t, string(getExpected), string(got))
		})
	}
}

func TestRendeFlagger(t *testing.T) {
	cases := []struct {
		name   string
		fleet  types.NamespacedName
		ref    *metav1.OwnerReference
		config *v1alpha1.FlaggerConfig
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
			config: &v1alpha1.FlaggerConfig{
				PublicTestloader:       true,
				TrafficRoutingProvider: v1alpha1.Istio,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderFlagger(manifestFS, tc.fleet, tc.ref, KubeConfigSecretRef{
				Name:       "cluster1",
				SecretName: "cluster1",
				SecretKey:  "kubeconfig.yaml",
			}, tc.config)
			assert.NoError(t, err)

			getExpected, err := getExpected("rollout", tc.name)
			assert.NoError(t, err)

			assert.Equal(t, string(getExpected), string(got))
		})
	}
}

func TestRendeRolloutTestloader(t *testing.T) {
	cases := []struct {
		name   string
		fleet  types.NamespacedName
		ref    *metav1.OwnerReference
		config *v1alpha1.FlaggerConfig
	}{
		{
			name: "testloader-default",
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
			config: &v1alpha1.FlaggerConfig{
				PublicTestloader:       true,
				TrafficRoutingProvider: v1alpha1.Istio,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderRolloutTestloader(manifestFS, tc.fleet, tc.ref, KubeConfigSecretRef{
				Name:       "cluster1",
				SecretName: "cluster1",
				SecretKey:  "kubeconfig.yaml",
			}, tc.config)
			assert.NoError(t, err)

			getExpected, err := getExpected("rollout", tc.name)
			assert.NoError(t, err)

			assert.Equal(t, string(getExpected), string(got))
		})
	}
}

func TestRenderSubmarinerBroker(t *testing.T) {
	cases := []struct {
		name   string
		fleet  types.NamespacedName
		ref    *metav1.OwnerReference
		config *v1alpha1.SubMarinerConfig
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
			config: &v1alpha1.SubMarinerConfig{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderSubmarinerBroker(manifestFS, tc.fleet, tc.ref, KubeConfigSecretRef{
				Name:       "cluster1",
				SecretName: "cluster1",
				SecretKey:  "kubeconfig.yaml",
			}, tc.config)
			assert.NoError(t, err)

			getExpected, err := getExpected("submariner-broker", tc.name)
			assert.NoError(t, err)
			assert.Equal(t, string(getExpected), string(got))
		})
	}
}

func TestRenderSubmarinerOperator(t *testing.T) {
	cases := []struct {
		name   string
		fleet  types.NamespacedName
		ref    *metav1.OwnerReference
		config *v1alpha1.SubMarinerConfig
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
			config: &v1alpha1.SubMarinerConfig{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderSubmarinerOperator(manifestFS, tc.fleet, tc.ref, KubeConfigSecretRef{
				Name:       "cluster1",
				SecretName: "cluster1",
				SecretKey:  "kubeconfig.yaml",
			}, tc.config)
			assert.NoError(t, err)

			getExpected, err := getExpected("submariner-operator", tc.name)
			assert.NoError(t, err)
			assert.Equal(t, string(getExpected), string(got))
		})
	}
}
