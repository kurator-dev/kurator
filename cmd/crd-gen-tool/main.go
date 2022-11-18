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

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"sync"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"helm.sh/helm/v3/pkg/kube"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/cli-runtime/pkg/resource"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/yaml"
)

var (
	addToScheme sync.Once
	nopLogger   = func(_ string, _ ...interface{}) {}
)

func newClient() *kube.Client {
	addToScheme.Do(func() {
		if err := certmanagerv1.AddToScheme(scheme.Scheme); err != nil {
			// This should never happen.
			panic(err)
		}
		if err := apiextv1.AddToScheme(scheme.Scheme); err != nil {
			// This should never happen.
			panic(err)
		}
		if err := apiextv1beta1.AddToScheme(scheme.Scheme); err != nil {
			panic(err)
		}
	})

	testFactory := cmdtesting.NewTestFactory()

	c := &kube.Client{
		Factory: testFactory.WithNamespace("default"),
		Log:     nopLogger,
	}

	return c
}

func main() {
	outputDir := env("OUTPUT_DIR", "manifests/charts/base/templates")
	clusterApiVersion := env("CLUSTER_API_PROVIDER_VERSION", "v1.2.5")
	awsProviderVersion := env("AWS_PROVIDER_VERSION", "v2.0.0")

	genCapi(outputDir, clusterApiVersion)
	genCapa(outputDir, awsProviderVersion)
}

func genCapi(outputDir string, version string) {
	fmt.Printf("start to gen Cluster API crds, version: %s output: %s \n", version, outputDir)
	infraComponentsYaml := fmt.Sprintf("https://github.com/kubernetes-sigs/cluster-api/releases/download/%s/core-components.yaml", version)
	resp, err := http.Get(infraComponentsYaml)
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		resp.Body.Close()
	}()

	c := newClient()
	resources, err := c.Build(resp.Body, false)
	if err != nil {
		fmt.Printf("build helm fail: %v", err)
		os.Exit(-1)
	}

	writeCRDs(outputDir, resources)
}

func genCapa(outputDir string, version string) {
	fmt.Printf("start to gen AWS crds, version: %s output: %s \n", version, outputDir)
	infraComponentsYaml := fmt.Sprintf("https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/%s/infrastructure-components.yaml", version)
	resp, err := http.Get(infraComponentsYaml)
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		resp.Body.Close()
	}()

	c := newClient()
	resources, err := c.Build(resp.Body, false)
	if err != nil {
		fmt.Printf("build helm fail: %v", err)
		os.Exit(-1)
	}

	writeCRDs(outputDir, resources)
}

func writeCRDs(outputDir string, resources kube.ResourceList) {
	crds := resources.Filter(func(r *resource.Info) bool {
		// only need CRD
		return r.Mapping.GroupVersionKind.Kind == "CustomResourceDefinition"
	})

	for _, r := range crds {
		out, _ := yaml.Marshal(r.Object)
		n := path.Join(outputDir, fmt.Sprintf("%s.yaml", r.Name))
		if err := os.WriteFile(n, out, 0o755); err != nil {
			fmt.Printf("write file err: %v", err)
			os.Exit(-1)
		}
	}
}

func env(key, defaultVal string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal
	}
	return v
}
