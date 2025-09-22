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

package tool

import (
	"os"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestPrintObj(t *testing.T) {
	component := make(map[string]componentElement)
	component["istio"] = componentElement{
		Name:             "istio",
		Cli:              "/home/root/.kurator/istio/1.13.3/istioctl",
		Hub:              "docker.io/istio",
		ReleaseURLPrefix: "https://github.com/istio/istio/releases/download",
		Version:          "1.13.3",
		Status:           "NotReady",
	}
	component["karmada"] = componentElement{
		Cli:              "/home/root/.kurator/karmada/v1.2.1/kubectl-karmada",
		Hub:              "",
		Name:             "karmada",
		ReleaseURLPrefix: "https://github.com/karmada-io/karmada/releases/download",
		Version:          "v1.2.1",
		Status:           "Ready",
	}

	cs, err := yaml.Marshal(component)
	if err != nil {
		panic(err)
	}

	w := &toolListWriter{
		cs: cs,
	}
	for _, v := range component {
		w.entry = append(w.entry, v)
	}

	if err := w.PrintObj(os.Stdout, Table.String()); err != nil {
		panic(err)
	}

	if err := w.PrintObj(os.Stdout, TableWIDE.String()); err != nil {
		panic(err)
	}

	if err := w.PrintObj(os.Stdout, YAML.String()); err != nil {
		panic(err)
	}

	if err := w.PrintObj(os.Stdout, JSON.String()); err != nil {
		panic(err)
	}
}
