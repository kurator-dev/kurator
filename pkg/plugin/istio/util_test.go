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

package istio

import (
	"os"
	"path"
	"testing"

	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	"github.com/stretchr/testify/assert"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"sigs.k8s.io/yaml"
)

func TestTemplateEastWest(t *testing.T) {
	c := clusterNetwork{
		MeshID:      "mesh1",
		ClusterName: "cluster1",
		Network:     "network1",
	}

	out, err := templateEastWest(c)
	assert.NoError(t, err)
	got := &iopv1alpha1.IstioOperator{}
	err = yaml.Unmarshal(out, got)
	assert.NoError(t, err)

	b, err := os.ReadFile(path.Join("testdata", "eastwest.yaml"))
	assert.NoError(t, err)
	expected := &iopv1alpha1.IstioOperator{}
	err = yaml.Unmarshal(b, expected)
	assert.NoError(t, err)

	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestTemplateNamespaceOverridePolicy(t *testing.T) {
	c := clusterNetwork{
		MeshID:               "mesh1",
		ClusterName:          "cluster1",
		IstioSystemNamespace: "istio-system",
		Network:              "network1",
	}

	out, err := templateIstioSystemOverridePolicy(c)
	assert.NoError(t, err)
	got := &policyv1alpha1.ClusterOverridePolicy{}
	err = yaml.Unmarshal(out, got)
	assert.NoError(t, err)

	b, err := os.ReadFile(path.Join("testdata", "istiosystem-overridepolicy.yaml"))
	assert.NoError(t, err)
	expected := &policyv1alpha1.ClusterOverridePolicy{}
	err = yaml.Unmarshal(b, expected)
	assert.NoError(t, err)

	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}
