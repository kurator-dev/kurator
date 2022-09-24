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

package istio

import (
	"bytes"
	"io/fs"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"kurator.dev/kurator/manifests"
	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/util"
)

const (
	eastEestTemplateFileName                           = "profiles/eastwest.tmpl.yaml"
	istiosystemNamespaceOverridePolicyTemplateFileName = "profiles/istio-namespace-override.tmpl.yaml"
	exposeServicesFileName                             = "profiles/expose-services.yaml"
)

func waitIngressgatewayReady(client *client.Client, opts *generic.Options, cluster string) error {
	return util.WaitMemberClusterPodReady(client, cluster, istioSystemNamespace, "app=istio-ingressgateway", opts.WaitInterval, opts.WaitTimeout)
}

func waitEastwestgatewayReady(client *client.Client, opts *generic.Options, cluster string) error {
	return util.WaitMemberClusterPodReady(client, cluster, istioSystemNamespace, "app=istio-eastwestgateway", opts.WaitInterval, opts.WaitTimeout)
}

func exposeServicesFiles() (string, error) {
	fsys := manifests.BuiltinOrDir("")
	out, err := fs.ReadFile(fsys, exposeServicesFileName)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

func templateEastWest(mesh clusterNetwork) ([]byte, error) {
	fsys := manifests.BuiltinOrDir("")
	gwTmpl, err := fs.ReadFile(fsys, eastEestTemplateFileName)
	if err != nil {
		return nil, err
	}

	return evaluate(string(gwTmpl), mesh)
}

func evaluate(text string, data interface{}) ([]byte, error) {
	t := template.New("istio template")
	tpl, err := t.Funcs(sprig.TxtFuncMap()).Parse(text)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := tpl.Execute(&b, data); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

type clusterNetwork struct {
	MeshID               string
	IstioSystemNamespace string
	ClusterName          string
	Network              string
}

func templateIstioSystemOverridePolicy(network clusterNetwork) ([]byte, error) {
	fsys := manifests.BuiltinOrDir("")
	gwTmpl, err := fs.ReadFile(fsys, istiosystemNamespaceOverridePolicyTemplateFileName)
	if err != nil {
		return nil, err
	}

	return evaluate(string(gwTmpl), network)
}
