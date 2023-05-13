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
	"bytes"
	"html/template"
	"io/fs"

	"github.com/Masterminds/sprig/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

type FleetPluginConfig struct {
	Name           string
	Component      string
	Fleet          types.NamespacedName
	OwnerReference *metav1.OwnerReference
	Cluster        *FleetCluster
	Chart          ChartConfig
	Values         map[string]interface{}
}

func (plugin FleetPluginConfig) ResourceName() string {
	if plugin.Cluster != nil {
		return plugin.Component + "-" + plugin.Cluster.Name
	}

	return plugin.Component
}

func (plugin FleetPluginConfig) StorageNamespace() string {
	// StorageNamespace is the namespace where the plugin stores its data
	// It's same as the target namespace for cluster scoped plugins
	if plugin.Cluster != nil {
		return plugin.Chart.TargetNamespace
	}

	return plugin.Fleet.Namespace
}

type FleetCluster struct {
	Name       string
	SecretName string
	SecretKey  string
}

type ChartConfig struct {
	Type            string
	Repo            string
	Name            string
	Version         string
	TargetNamespace string
	Values          map[string]interface{}
}

func getFleetPluginChart(fsys fs.FS, pluginName string) (*ChartConfig, error) {
	out, err := fs.ReadFile(fsys, "profiles/fleet/plugins/"+pluginName+".yaml")
	if err != nil {
		return nil, err
	}

	var c ChartConfig
	if err := yaml.Unmarshal(out, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

func renderFleetPlugin(fsys fs.FS, cfg FleetPluginConfig) ([]byte, error) {
	out, err := fs.ReadFile(fsys, "profiles/fleet/plugin.tpl")
	if err != nil {
		return nil, err
	}

	t := template.New("fleet plugin template")
	tpl, err := t.Funcs(funMap()).Parse(string(out))
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := tpl.Execute(&b, cfg); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func funMap() template.FuncMap {
	m := sprig.TxtFuncMap()
	m["toYaml"] = toYaml
	return m
}

func toYaml(value interface{}) string {
	y, err := yaml.Marshal(value)
	if err != nil {
		return ""
	}

	return string(y)
}
