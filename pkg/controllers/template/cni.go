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

package template

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"kurator.dev/kurator/pkg/controllers/scope"
)

//go:embed cni.yaml.tpl
var cniTpl string

//go:embed calico.yaml
var calicoYaml string

type cniOptions struct {
	scope.CNI

	CNIYaml string
}

func RenderCNI(cni scope.CNI) ([]byte, error) {
	switch cni.Type {
	case "calico":
		opts := &cniOptions{
			CNI:     cni,
			CNIYaml: calicoYaml,
		}
		return renderCNI(opts)
	default:
		return nil, fmt.Errorf("unknown CNI type: %s", cni.Type)
	}

}

func renderCNI(c *cniOptions) ([]byte, error) {
	t := template.New("cni template")
	tpl, err := t.Funcs(sprig.TxtFuncMap()).Parse(cniTpl)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := tpl.Execute(&b, c); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
