/*
Copyright 2022-2025 Kurator Authors.

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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"kurator.dev/kurator/pkg/fleet-manager/manifests"
)

var manifestFS = manifests.BuiltinOrDir("")

func TestGetFleetPluginChart(t *testing.T) {
	cases := []struct {
		name     string
		expected *ChartConfig
	}{
		{
			name: "thanos",
			expected: &ChartConfig{
				Type:            "oci",
				Repo:            "oci://registry-1.docker.io/bitnamicharts",
				Name:            "thanos",
				Version:         "12.5.1",
				TargetNamespace: "monitoring",
				Values: map[string]interface{}{
					"query": map[string]interface{}{
						"dnsDiscovery": map[string]interface{}{
							"sidecarsService":   "thanos-sidecar-remote",
							"sidecarsNamespace": "monitoring",
						},
					},
					"queryFrontend": map[string]interface{}{
						"enabled": false,
					},
					"bucketweb": map[string]interface{}{
						"enabled": false,
					},
					"compactor": map[string]interface{}{
						"enabled": false,
					},
					"storegateway": map[string]interface{}{
						"enabled": true,
					},
					"ruler": map[string]interface{}{
						"enabled": false,
					},
					"metrics": map[string]interface{}{
						"enabled": false,
					},
					"minio": map[string]interface{}{
						"enabled": false,
					},
				},
			},
		},
		{
			name: "prometheus",
			expected: &ChartConfig{
				Type:            "oci",
				Repo:            "oci://registry-1.docker.io/bitnamicharts",
				Name:            "kube-prometheus",
				Version:         "8.9.1",
				TargetNamespace: "monitoring",
				Values: map[string]interface{}{
					"fullnameOverride": "prometheus",
					"alertmanager": map[string]interface{}{
						"enabled": false,
					},
					"exporters": map[string]interface{}{
						"enabled": false,
						"node-exporter": map[string]interface{}{
							"enabled": false,
						},
						"kube-state-metrics": map[string]interface{}{
							"enabled": false,
						},
					},
					"blackboxExporter": map[string]interface{}{
						"enabled": false,
					},
					"operator": map[string]interface{}{
						"service": map[string]interface{}{
							"type": "ClusterIP",
						},
					},
					"prometheus": map[string]interface{}{
						"disableCompaction": true,
						"thanos": map[string]interface{}{
							"create": true,
							"service": map[string]interface{}{
								"type": "LoadBalancer",
							},
							"objectStorageConfig": map[string]interface{}{
								"secretKey":  "objstore.yml",
								"secretName": "thanos-objstore",
							},
						},
						"service": map[string]interface{}{
							"type": "ClusterIP",
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getFleetPluginChart(manifestFS, tc.name)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func getExpected(plugin, casename string) ([]byte, error) {
	return os.ReadFile("testdata/" + plugin + "/" + casename + ".yaml")
}
