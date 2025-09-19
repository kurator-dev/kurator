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
	"encoding/json"
	"fmt"
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
	MetricPluginName             = "metric"
	GrafanaPluginName            = "grafana"
	KyvernoPluginName            = "kyverno"
	BackupPluginName             = "backup"
	StorageOperatorPluginName    = "storage-operator"
	ClusterStoragePluginName     = "cluster-storage"
	FlaggerPluginName            = "flagger"
	PublicTestloaderName         = "testloader"
	SubMarinerBrokerPluginName   = "submariner-broker"
	SubMarinerOperatorPluginName = "submariner-operator"

	ThanosComponentName             = "thanos"
	PrometheusComponentName         = "prometheus"
	GrafanaComponentName            = "grafana"
	KyvernoComponentName            = "kyverno"
	KyvernoPolicyComponentName      = "kyverno-policies"
	VeleroComponentName             = "velero"
	RookOperatorComponentName       = "rook"
	RookClusterComponentName        = "rook-ceph"
	FlaggerComponentName            = "flagger"
	TestloaderComponentName         = "testloader"
	SubMarinerBrokerComponentName   = "sm-broker"
	SubMarinerOperatorComponentName = "sm-operator"

	OCIReposiotryPrefix = "oci://"
)

var ProviderNamespace = map[fleetv1a1.Provider]string{
	"istio": "istio-system",
	"kuma":  "kuma-system",
	"nginx": "ingress-nginx",
}

type GrafanaDataSource struct {
	Name       string `json:"name"`
	SourceType string `json:"type"`
	URL        string `json:"url"`
	Access     string `json:"access"`
	IsDefault  bool   `json:"isDefault"`
}

func RenderKyvernoPolicy(fsys fs.FS, fleetNN types.NamespacedName, fleetRef *metav1.OwnerReference, cluster KubeConfigSecretRef, kyvernoCfg *fleetv1a1.KyvernoConfig) ([]byte, error) {
	c, err := getFleetPluginChart(fsys, KyvernoPolicyComponentName)
	if err != nil {
		return nil, err
	}

	mergeChartConfig(c, kyvernoCfg.Chart)

	values := map[string]interface{}{
		"podSecurityStandard":     kyvernoCfg.PodSecurity.Standard,
		"podSecuritySeverity":     kyvernoCfg.PodSecurity.Severity,
		"validationFailureAction": kyvernoCfg.PodSecurity.ValidationFailureAction,
	}

	return renderFleetPlugin(fsys, FleetPluginConfig{
		Name:           KyvernoPluginName,
		Component:      KyvernoPolicyComponentName,
		Fleet:          fleetNN,
		Cluster:        &cluster,
		OwnerReference: fleetRef,
		Chart:          *c,
		Values:         values,
	})
}

func RenderKyverno(fsys fs.FS, fleetNN types.NamespacedName, fleetRef *metav1.OwnerReference, cluster KubeConfigSecretRef, kyvernoCfg *fleetv1a1.KyvernoConfig) ([]byte, error) {
	c, err := getFleetPluginChart(fsys, KyvernoComponentName)
	if err != nil {
		return nil, err
	}

	mergeChartConfig(c, kyvernoCfg.Chart)

	values, err := toMap(kyvernoCfg.ExtraArgs)
	if err != nil {
		return nil, err
	}

	return renderFleetPlugin(fsys, FleetPluginConfig{
		Name:           KyvernoPluginName,
		Component:      KyvernoComponentName,
		Fleet:          fleetNN,
		Cluster:        &cluster,
		OwnerReference: fleetRef,
		Chart:          *c,
		Values:         values,
	})
}

func RenderGrafana(fsys fs.FS, fleetNN types.NamespacedName, fleetRef *metav1.OwnerReference,
	grafanaCfg *fleetv1a1.GrafanaConfig, datasources []*GrafanaDataSource) ([]byte, error) {
	c, err := getFleetPluginChart(fsys, GrafanaComponentName)
	if err != nil {
		return nil, err
	}

	mergeChartConfig(c, grafanaCfg.Chart)
	c.TargetNamespace = fleetNN.Namespace // grafana chart is fleet scoped

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

func RenderPrometheus(fsys fs.FS, fleetName types.NamespacedName, fleetRef *metav1.OwnerReference, cluster KubeConfigSecretRef, metricCfg *fleetv1a1.MetricConfig) ([]byte, error) {
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

type veleroObjectStoreLocation struct {
	Bucket   string                 `json:"bucket"`
	Provider string                 `json:"provider"`
	Config   map[string]interface{} `json:"config"`
}

func RenderVelero(
	fsys fs.FS,
	fleetNN types.NamespacedName,
	fleetRef *metav1.OwnerReference,
	cluster KubeConfigSecretRef,
	backupCfg *fleetv1a1.BackupConfig,
	veleroSecretName string,
) ([]byte, error) {
	// get and merge the chart config
	c, err := getFleetPluginChart(fsys, VeleroComponentName)
	if err != nil {
		return nil, err
	}
	mergeChartConfig(c, backupCfg.Chart)

	// get default values
	defaultValues := c.Values
	// providerValues is a map that stores default configurations associated with the specific provider. These configurations are necessary for the proper functioning of the Velero tool with the provider. Currently, this includes configurations for initContainers.
	providerValues, err := getProviderValues(backupCfg.Storage.Location.Provider)
	if err != nil {
		return nil, err
	}
	// add providerValues to default values
	defaultValues = transform.MergeMaps(defaultValues, providerValues)

	// get custom values
	customValues := map[string]interface{}{}
	locationConfig := stringMapToInterfaceMap(backupCfg.Storage.Location.Config)
	// generate velero config. "backupCfg.Storage.Location.Endpoint" and "backupCfg.Storage.Location.Region" will overwrite the value of "backupCfg.Storage.Location.config"
	// because "backupCfg.Storage.Location.config" is optional, it should take effect only when current setting is not enough.
	Config := transform.MergeMaps(locationConfig, map[string]interface{}{
		"s3Url":            backupCfg.Storage.Location.Endpoint,
		"region":           backupCfg.Storage.Location.Region,
		"s3ForcePathStyle": true,
	})
	provider := getProviderFrombackupCfg(backupCfg)
	configurationValues := map[string]interface{}{
		"configuration": map[string]interface{}{
			"backupStorageLocation": []veleroObjectStoreLocation{
				{
					Bucket:   backupCfg.Storage.Location.Bucket,
					Provider: provider,
					Config:   Config,
				},
			},
		},
		"credentials": map[string]interface{}{
			"useSecret":      true,
			"existingSecret": veleroSecretName,
		},
	}
	// add custom configurationValues to customValues
	customValues = transform.MergeMaps(customValues, configurationValues)
	extraValues, err := toMap(backupCfg.ExtraArgs)
	if err != nil {
		return nil, err
	}
	// add custom extraValues to customValues
	customValues = transform.MergeMaps(customValues, extraValues)

	// replace the default values with custom values to obtain the actual values.
	values := transform.MergeMaps(defaultValues, customValues)

	return renderFleetPlugin(fsys, FleetPluginConfig{
		Name:           BackupPluginName,
		Component:      VeleroComponentName,
		Fleet:          fleetNN,
		Cluster:        &cluster,
		OwnerReference: fleetRef,
		Chart:          *c,
		Values:         values,
	})
}

// RenderStorageOperator builds configuration of the rendering rook-operator.
func RenderStorageOperator(
	fsys fs.FS,
	fleetNN types.NamespacedName,
	fleetRef *metav1.OwnerReference,
	cluster KubeConfigSecretRef,
	distributedStorageCfg *fleetv1a1.DistributedStorageConfig,
) ([]byte, error) {
	// get and merge the chart config
	c, err := getFleetPluginChart(fsys, RookOperatorComponentName)
	if err != nil {
		return nil, err
	}
	mergeChartConfig(c, distributedStorageCfg.Chart)

	values, err := toMap(distributedStorageCfg.ExtraArgs)
	if err != nil {
		return nil, err
	}

	return renderFleetPlugin(fsys, FleetPluginConfig{
		Name:           StorageOperatorPluginName,
		Component:      RookOperatorComponentName,
		Fleet:          fleetNN,
		Cluster:        &cluster,
		OwnerReference: fleetRef,
		Chart:          *c,
		Values:         values,
	})
}

// Build configuration of the rendering rook-ceph-cluster.
func RenderClusterStorage(
	fsys fs.FS,
	fleetNN types.NamespacedName,
	fleetRef *metav1.OwnerReference,
	cluster KubeConfigSecretRef,
	distributedStorageCfg *fleetv1a1.DistributedStorageConfig,
) ([]byte, error) {
	c, err := getFleetPluginChart(fsys, RookClusterComponentName)
	if err != nil {
		return nil, err
	}
	mergeChartConfig(c, distributedStorageCfg.Chart)

	// get default values
	defaultValues := c.Values
	// In the rook, the Labels, annotation and Placement of Monitor and manager are configured under the Labels, annotation and Placement fields.
	// So it need to be rebuild customValues using user settings in distributedStorage.

	customValues := buildStorageClusterValue(*distributedStorageCfg)
	cephClusterValue := make(map[string]interface{})
	cephClusterValue["cephClusterSpec"] = customValues
	extraValues, err := toMap(distributedStorageCfg.ExtraArgs)
	if err != nil {
		return nil, err
	}
	// Add custom extraValues to cephClusterValue.
	cephClusterValue = transform.MergeMaps(cephClusterValue, extraValues)
	// Replace the default values with custom values to obtain the actual values.
	values := transform.MergeMaps(defaultValues, cephClusterValue)

	return renderFleetPlugin(fsys, FleetPluginConfig{
		Name:           ClusterStoragePluginName,
		Component:      RookClusterComponentName,
		Fleet:          fleetNN,
		Cluster:        &cluster,
		OwnerReference: fleetRef,
		Chart:          *c,
		Values:         values,
	})
}

func RenderFlagger(
	fsys fs.FS,
	fleetNN types.NamespacedName,
	fleetRef *metav1.OwnerReference,
	cluster KubeConfigSecretRef,
	flaggerConfig *fleetv1a1.FlaggerConfig,
) ([]byte, error) {
	// get and merge the chart config
	c, err := getFleetPluginChart(fsys, FlaggerComponentName)
	if err != nil {
		return nil, err
	}
	mergeChartConfig(c, flaggerConfig.Chart)
	c.TargetNamespace = ProviderNamespace[flaggerConfig.TrafficRoutingProvider]

	values, err := toMap(flaggerConfig.ExtraArgs)
	if flaggerConfig.TrafficRoutingProvider == fleetv1a1.Nginx {
		values = transform.MergeMaps(values, map[string]interface{}{
			"prometheus": map[string]interface{}{
				"install": true,
			},
			"meshProvider": "nginx",
		})
	}
	if err != nil {
		return nil, err
	}

	return renderFleetPlugin(fsys, FleetPluginConfig{
		Name:           FlaggerPluginName,
		Component:      FlaggerComponentName,
		Fleet:          fleetNN,
		Cluster:        &cluster,
		OwnerReference: fleetRef,
		Chart:          *c,
		Values:         values,
	})
}

func RenderProvider(
	fsys fs.FS,
	fleetNN types.NamespacedName,
	fleetRef *metav1.OwnerReference,
	cluster KubeConfigSecretRef,
	flaggerConfig *fleetv1a1.FlaggerConfig,
) ([]byte, error) {
	name := string(flaggerConfig.TrafficRoutingProvider)
	// get and merge the chart config
	c, err := getFleetPluginChart(fsys, name)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{}
	if providerConfig := flaggerConfig.ProviderConfig; providerConfig != nil {
		mergeChartConfig(c, providerConfig.Chart)
		values, err = toMap(providerConfig.ExtraArgs)
		if err != nil {
			return nil, err
		}
	}
	c.TargetNamespace = ProviderNamespace[flaggerConfig.TrafficRoutingProvider]

	return renderFleetPlugin(fsys, FleetPluginConfig{
		Name:           name,
		Component:      name,
		Fleet:          fleetNN,
		Cluster:        &cluster,
		OwnerReference: fleetRef,
		Chart:          *c,
		Values:         values,
	})
}

func RenderRolloutTestloader(
	fsys fs.FS,
	fleetNN types.NamespacedName,
	fleetRef *metav1.OwnerReference,
	cluster KubeConfigSecretRef,
	flaggerConfig *fleetv1a1.FlaggerConfig,
) ([]byte, error) {
	// get and merge the chart config
	c, err := getFleetPluginChart(fsys, TestloaderComponentName)
	if err != nil {
		return nil, err
	}
	// Installed in the same namespace as flagger.
	c.TargetNamespace = ProviderNamespace[flaggerConfig.TrafficRoutingProvider]
	// make sure use the specified testlaoder.
	mergeChartConfig(c, nil)
	values, err := toMap(flaggerConfig.ExtraArgs)
	if err != nil {
		return nil, err
	}

	return renderFleetPlugin(fsys, FleetPluginConfig{
		Name:           PublicTestloaderName,
		Component:      TestloaderComponentName,
		Fleet:          fleetNN,
		Cluster:        &cluster,
		OwnerReference: fleetRef,
		Chart:          *c,
		Values:         values,
	})
}

func RenderSubmarinerBroker(
	fsys fs.FS,
	fleetNN types.NamespacedName,
	fleetRef *metav1.OwnerReference,
	cluster KubeConfigSecretRef,
) ([]byte, error) {
	c, err := getFleetPluginChart(fsys, SubMarinerBrokerComponentName)
	if err != nil {
		return nil, err
	}

	return renderFleetPlugin(fsys, FleetPluginConfig{
		Name:           SubMarinerBrokerPluginName,
		Component:      SubMarinerBrokerComponentName,
		Fleet:          fleetNN,
		Cluster:        &cluster,
		OwnerReference: fleetRef,
		Chart:          *c,
	})
}

func RenderSubmarinerOperator(
	fsys fs.FS,
	fleetNN types.NamespacedName,
	fleetRef *metav1.OwnerReference,
	cluster KubeConfigSecretRef,
	subMarinerOperatorConfig *fleetv1a1.SubMarinerOperatorConfig,
	brokerConfig map[string]interface{},
) ([]byte, error) {
	// get and merge the chart config
	c, err := getFleetPluginChart(fsys, SubMarinerOperatorComponentName)
	if err != nil {
		return nil, err
	}
	mergeChartConfig(c, subMarinerOperatorConfig.Chart)

	values, err := toMap(subMarinerOperatorConfig.ExtraArgs)
	if err != nil {
		return nil, err
	}
	globalnet := false
	globalCidr, ok := subMarinerOperatorConfig.Globalcidrs[cluster.Name]
	if ok && globalCidr != "" {
		globalnet = true
	}

	brokerConfig["globalnet"] = globalnet

	values = transform.MergeMaps(values, map[string]interface{}{
		"broker": brokerConfig,
		"serviceAccounts": map[string]interface{}{
			"globalnet": map[string]interface{}{
				"create": globalnet,
			},
		},
		"submariner": map[string]interface{}{
			"clusterId":   cluster.Name,
			"clusterCidr": subMarinerOperatorConfig.ClusterCidrs[cluster.Name],
			"serviceCidr": subMarinerOperatorConfig.ServiceCidrs[cluster.Name],
			"globalCidr":  globalCidr,
		},
	})

	return renderFleetPlugin(fsys, FleetPluginConfig{
		Name:           SubMarinerOperatorPluginName,
		Component:      SubMarinerOperatorComponentName,
		Fleet:          fleetNN,
		Cluster:        &cluster,
		OwnerReference: fleetRef,
		Chart:          *c,
		Values:         values,
	})
}

// According to distributedStorageCfg, generate the configuration for rook-ceph
func buildStorageClusterValue(distributedStorageCfg fleetv1a1.DistributedStorageConfig) map[string]interface{} {
	customValues := make(map[string]interface{})
	if distributedStorageCfg.Storage.DataDirHostPath != nil {
		customValues["dataDirHostPath"] = distributedStorageCfg.Storage.DataDirHostPath
	}
	if distributedStorageCfg.Storage.Storage != nil {
		customValues["storage"] = distributedStorageCfg.Storage.Storage
	}
	if distributedStorageCfg.Storage.Monitor != nil {
		monitorCfg := distributedStorageCfg.Storage.Monitor
		if monitorCfg.Count != nil {
			monitorMap := make(map[string]interface{})
			monitorMap["count"] = monitorCfg.Count
			customValues["mon"] = monitorMap
		}
		if monitorCfg.Labels != nil {
			_, ok := customValues["labels"]
			if !ok {
				labelsMap := make(map[string]interface{})
				labelsMap["mon"] = monitorCfg.Labels
				customValues["labels"] = labelsMap
			} else {
				labelsMap := customValues["labels"].(map[string]interface{})
				labelsMap["mon"] = monitorCfg.Labels
				customValues["labels"] = labelsMap
			}
		}
		if monitorCfg.Annotations != nil {
			_, ok := customValues["annotations"]
			if !ok {
				annotationsMap := make(map[string]interface{})
				annotationsMap["mon"] = monitorCfg.Annotations
				customValues["annotations"] = annotationsMap
			} else {
				annotationsMap := customValues["annotations"].(map[string]interface{})
				annotationsMap["mon"] = monitorCfg.Annotations
				customValues["annotations"] = annotationsMap
			}
		}
		if monitorCfg.Placement != nil {
			_, ok := customValues["placement"]
			if !ok {
				placementMap := make(map[string]interface{})
				placementMap["mon"] = monitorCfg.Placement
				customValues["placement"] = placementMap
			} else {
				placementMap := customValues["placement"].(map[string]interface{})
				placementMap["mon"] = monitorCfg.Placement
				customValues["placement"] = placementMap
			}
		}
	}
	if distributedStorageCfg.Storage.Manager != nil {
		managerCfg := distributedStorageCfg.Storage.Manager
		if managerCfg.Count != nil {
			managerMap := make(map[string]interface{})
			managerMap["count"] = managerCfg.Count
			customValues["mgr"] = managerMap
		}
		if managerCfg.Labels != nil {
			_, ok := customValues["labels"]
			if !ok {
				labelsMap := make(map[string]interface{})
				labelsMap["mgr"] = managerCfg.Labels
				customValues["labels"] = labelsMap
			} else {
				labelsMap := customValues["labels"].(map[string]interface{})
				labelsMap["mgr"] = managerCfg.Labels
				customValues["labels"] = labelsMap
			}
		}
		if managerCfg.Annotations != nil {
			_, ok := customValues["annotations"]
			if !ok {
				annotationsMap := make(map[string]interface{})
				annotationsMap["mgr"] = managerCfg.Annotations
				customValues["annotations"] = annotationsMap
			} else {
				annotationsMap := customValues["annotations"].(map[string]interface{})
				annotationsMap["mgr"] = managerCfg.Annotations
				customValues["annotations"] = annotationsMap
			}
		}
		if managerCfg.Placement != nil {
			_, ok := customValues["placement"]
			if !ok {
				placementMap := make(map[string]interface{})
				placementMap["mgr"] = managerCfg.Placement
				customValues["placement"] = placementMap
			} else {
				placementMap := customValues["placement"].(map[string]interface{})
				placementMap["mgr"] = managerCfg.Placement
				customValues["placement"] = placementMap
			}
		}
	}
	return customValues
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

func stringMapToInterfaceMap(args map[string]string) map[string]interface{} {
	m := make(map[string]interface{})
	for s, s2 := range args {
		m[s] = s2
	}

	return m
}

// getProviderValues return the map that stores default configurations associated with the specific provider.
// The provider parameter can be one of the following values: "aws", "huaweicloud", "gcp", "azure".
func getProviderValues(provider string) (map[string]interface{}, error) {
	switch provider {
	case "aws":
		return buildAWSProviderValues(), nil
	case "huaweicloud":
		return buildHuaWeiCloudProviderValues(), nil
	case "gcp":
		return buildGCPProviderValues(), nil
	case "azure":
		return buildAzureProviderValues(), nil
	default:
		return nil, fmt.Errorf("unknown objStoreProvider: %v", provider)
	}
}

// buildAWSProviderValues constructs the default provider values for AWS.
func buildAWSProviderValues() map[string]interface{} {
	values := map[string]interface{}{}

	// currently, the default provider-related extra configuration only sets up initContainers
	initContainersConfig := map[string]interface{}{
		"initContainers": []interface{}{
			map[string]interface{}{
				"image": "velero/velero-plugin-for-aws:v1.7.1",
				"name":  "velero-plugin-for-aws",
				"volumeMounts": []interface{}{
					map[string]interface{}{
						"mountPath": "/target",
						"name":      "plugins",
					},
				},
			},
		},
	}
	values = transform.MergeMaps(values, initContainersConfig)

	return values
}

func buildHuaWeiCloudProviderValues() map[string]interface{} {
	return buildAWSProviderValues()
}

// TODO： accomplish those function after investigation
func buildGCPProviderValues() map[string]interface{} {
	return nil
}
func buildAzureProviderValues() map[string]interface{} {
	return nil
}

func getProviderFrombackupCfg(backupCfg *fleetv1a1.BackupConfig) string {
	provider := backupCfg.Storage.Location.Provider
	// there no "huaweicloud" provider in velero
	if provider == "huaweicloud" {
		provider = "aws"
	}
	return provider
}
