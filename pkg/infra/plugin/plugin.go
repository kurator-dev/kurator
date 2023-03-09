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
	_ "embed"
	"fmt"
	"io/fs"
	"path"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"

	"kurator.dev/kurator/manifests"
	"kurator.dev/kurator/pkg/infra/scope"
)

const (
	pluginBasePath = "profiles/infra/plugins"
)

// pluginYAMLFunc returns the plugin yaml, only test can override this function to return test data
var pluginYAMLFunc = getPluginYAML

//go:embed plugin.yaml.tpl
var pluginTpl string

type PluginOptions struct {
	ClusterName string
	Name        string
	Namespace   string
	PluginYAML  string
}

func RenderCNI(cluster *scope.Cluster) ([]byte, error) {
	opts := &PluginOptions{
		Name:        fmt.Sprintf("%s-%s-cni", cluster.Name, cluster.UID),
		ClusterName: cluster.Name,
		Namespace:   cluster.Namespace,
	}

	pluginFilename := path.Join(pluginBasePath, fmt.Sprintf("%s-cni-%s.yaml", cluster.InfraType, cluster.CNIType))
	out, err := pluginYAMLFunc(pluginFilename)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cni yaml form %s", pluginFilename)
	}

	t := template.New("plugin template")
	tpl, err := t.Funcs(sprig.TxtFuncMap()).Parse(out)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := tpl.Execute(&b, cluster.Cluster.Spec); err != nil {
		return nil, err
	}

	opts.PluginYAML = b.String()
	return Render(opts)
}

func getPluginYAML(f string) (string, error) {
	fsys := manifests.BuiltinOrDir("")
	out, err := fs.ReadFile(fsys, f)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func Render(opts *PluginOptions) ([]byte, error) {
	t := template.New("plugin template")
	tpl, err := t.Funcs(sprig.TxtFuncMap()).Parse(pluginTpl)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := tpl.Execute(&b, opts); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
