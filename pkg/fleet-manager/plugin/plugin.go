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
	"encoding/json"
	"io/fs"
	"strings"

	"github.com/fluxcd/pkg/runtime/transform"
	sourcev1b2 "github.com/fluxcd/source-controller/api/v1beta2"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	fleetv1a1 "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

const (
	MetricPluginName  = "metric"
	GrafanaPluginName = "grafana"

	ThanosComponentName     = "thanos"
	PrometheusComponentName = "prometheus"
	GrafanaComponentName    = "grafana"

	OCIReposiotryPrefix = "oci://"
)

type GrafanaDataSource struct {
	Name       string `json:"name"`
	SourceType string `json:"type"`
	URL        string `json:"url"`
	Access     string `json:"access"`
	IsDefault  bool   `json:"isDefault"`
}

func RenderGrafana(fsys fs.FS, fleetNN types.NamespacedName, fleetRef *metav1.OwnerReference, grafanaCfg *fleetv1a1.GrafanaConfig, datasources []*GrafanaDataSource) ([]byte, error) {
	c, err := getFleetPluginChart(fsys, GrafanaComponentName)
	if err != nil {
		return nil, err
	}

	mergeChartConfig(c, grafanaCfg.Chart)
	c.TargetNamespace = fleetNN.Namespace // thanos chart is fleet scoped

	values, err := toMap(grafanaCfg.ExtraArgs)
	if err != nil {
		return nil, err
	}

	if len(datasources) != 0 {
		values = transform.MergeMaps(values, map[string]interface{}{
			"datasources": map[string]interface{}{
				"secretDefinition": map[string]interface{}{
					"apiVersion":  1,
					"datasources": datasources,
				},
			},
		})
	}

	grafanaPluginCfg := FleetPluginConfig{
		Name:           GrafanaPluginName,
		Component:      GrafanaComponentName,
		Fleet:          fleetNN,
		OwnerReference: fleetRef,
		Chart:          *c,
		Values:         values,
	}

	return renderFleetPlugin(fsys, grafanaPluginCfg)
}

func RenderThanos(fsys fs.FS, fleetNN types.NamespacedName, fleetRef *metav1.OwnerReference, metricCfg *fleetv1a1.MetricConfig) ([]byte, error) {
	thanosChart, err := getFleetPluginChart(fsys, ThanosComponentName)
	if err != nil {
		return nil, err
	}

	mergeChartConfig(thanosChart, metricCfg.Thanos.Chart)
	thanosChart.TargetNamespace = fleetNN.Namespace // thanos chart is fleet scoped

	values, err := toMap(metricCfg.Thanos.ExtraArgs)
	if err != nil {
		return nil, err
	}

	values = transform.MergeMaps(values, map[string]interface{}{
		"existingObjstoreSecret": metricCfg.Thanos.ObjectStoreConfig.SecretName, // always use secret from API
		"query": map[string]interface{}{
			"dnsDiscovery": map[string]interface{}{
				"sidecarsNamespace": fleetNN.Namespace, // override default namespace
			},
		},
	})

	thanosCfg := FleetPluginConfig{
		Name:           MetricPluginName,
		Component:      ThanosComponentName,
		Fleet:          fleetNN,
		OwnerReference: fleetRef,
		Chart:          *thanosChart,
		Values:         values,
	}

	return renderFleetPlugin(fsys, thanosCfg)
}

func RendPrometheus(fsys fs.FS, fleetName types.NamespacedName, fleetRef *metav1.OwnerReference, cluster FleetCluster, metricCfg *fleetv1a1.MetricConfig) ([]byte, error) {
	promChart, err := getFleetPluginChart(fsys, PrometheusComponentName)
	if err != nil {
		return nil, err
	}

	mergeChartConfig(promChart, metricCfg.Prometheus.Chart)

	values, err := toMap(metricCfg.Prometheus.ExtraArgs)
	if err != nil {
		return nil, err
	}

	values = transform.MergeMaps(values, map[string]interface{}{
		"prometheus": map[string]interface{}{
			"externalLabels": map[string]interface{}{
				"cluster": cluster.Name,
			},
			"thanos": map[string]interface{}{
				"objectStorageConfig": map[string]interface{}{
					"secretName": metricCfg.Thanos.ObjectStoreConfig.SecretName,
				},
			},
		},
	})

	promCfg := FleetPluginConfig{
		Name:           MetricPluginName,
		Component:      PrometheusComponentName,
		Fleet:          fleetName,
		OwnerReference: fleetRef,
		Cluster:        &cluster,
		Chart:          *promChart,
		Values:         values,
	}

	return renderFleetPlugin(fsys, promCfg)
}

func mergeChartConfig(origin *ChartConfig, target *fleetv1a1.ChartConfig) {
	if target == nil {
		return
	}

	origin.Name = target.Name
	origin.Repo = target.Repository
	origin.Version = target.Version
	if target.Repository != "" &&
		strings.HasPrefix(target.Repository, OCIReposiotryPrefix) {
		origin.Type = sourcev1b2.HelmRepositoryTypeOCI
	} else {
		origin.Type = sourcev1b2.HelmRepositoryTypeDefault
	}
}

func toMap(args apiextensionsv1.JSON) (map[string]interface{}, error) {
	if args.Raw == nil {
		return nil, nil
	}

	var m map[string]interface{}
	err := json.Unmarshal(args.Raw, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
