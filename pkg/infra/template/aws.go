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
	"io/fs"
	"path"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"

	"kurator.dev/kurator/manifests"
	"kurator.dev/kurator/pkg/infra/scope"
)

const (
	tplBasePath = "profiles/infra/capitpls"
)

func RenderClusterAPIForAWS(c *scope.Cluster) ([]byte, error) {
	tplText, err := getTemplate(c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get template for Infra %s", c.InfraType)
	}

	t := template.New("aws template")
	tpl, err := t.Funcs(sprig.TxtFuncMap()).Parse(tplText)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	if err := tpl.Execute(&b, c); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func getTemplate(c *scope.Cluster) (string, error) {
	f := path.Join(tplBasePath, fmt.Sprintf("%s.yaml.tpl", c.InfraType))
	fsys := manifests.BuiltinOrDir("")
	out, err := fs.ReadFile(fsys, f)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
