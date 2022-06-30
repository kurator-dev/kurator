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

package generic

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/mitchellh/cli"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/yaml"

	"kurator.dev/kurator/manifests"
)

type Options struct {
	DryRun bool // TODO: support dry run

	config *genericclioptions.ConfigFlags

	Ui cli.Ui
	// HomeDir is an absolute path which most importantly contains "versions" installed from binary. Defaults to DefaultHomeDir
	HomeDir string

	// The interval and timeout used to check installation status.
	WaitInterval time.Duration
	WaitTimeout  time.Duration

	KubeConfig  string
	KubeContext string

	Components map[string]Component
}

func New() *Options {
	g := &Options{
		Ui: &cli.BasicUi{
			Writer:      os.Stdout,
			ErrorWriter: os.Stdout,
			Reader:      os.Stdin,
		},
	}
	g.Components = loadComponents(g.HomeDir)
	// bind to kubernetes config flags
	g.config = &genericclioptions.ConfigFlags{
		Context:    &g.KubeContext,
		KubeConfig: &g.KubeConfig,
	}
	return g
}

// AddFlags binds flags to the given flagset.
func (g *Options) AddFlags(fs *pflag.FlagSet) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}

	fs.StringVar(&g.HomeDir, "home-dir", path.Join(homeDir, ".kurator"), "install path, default to $HOME/.kurator")

	fs.StringVarP(&g.KubeConfig, "kubeconfig", "c", "/etc/karmada/karmada-apiserver.config", "path to the kubeconfig file, default to karmada apiserver config")
	fs.StringVar(&g.KubeContext, "context", "", "name of the kubeconfig context to use")

	fs.BoolVar(&g.DryRun, "dry-run", false, "console/log output only, make no changes.")

	fs.DurationVar(&g.WaitInterval, "wait-interval", 1*time.Second, "interval used for checking pod ready, default value is 1s.")
	fs.DurationVar(&g.WaitTimeout, "wait-timeout", 2*time.Minute, "timeout used for checking pod ready, default value is 2m.")
}

// RESTClientGetter gets the kubeconfig from EnvSettings
func (g *Options) RESTClientGetter() genericclioptions.RESTClientGetter {
	return g.config
}

type cfg struct {
	Components []Component `json:"components"`
}

type Component struct {
	Name             string `yaml:"name"`
	Version          string `yaml:"version"`
	Hub              string `yaml:"hub"`
	ReleaseURLPrefix string `yaml:"releaseURLPrefix"`
}

const (
	componentsYaml        = "components.yaml"
	builtinComponentsYaml = "profiles/" + componentsYaml
)

func loadComponents(homeDir string) map[string]Component {
	b := loadComponentsYaml(homeDir)
	var c cfg
	if err := yaml.Unmarshal(b, &c); err != nil {
		logrus.Fatalf("failed ummarshal components: %v", err)
	}

	components := make(map[string]Component, len(c.Components))
	for _, com := range c.Components {
		components[com.Name] = com
	}

	logrus.Debugf("components: %+v", components)
	return components
}

func loadComponentsYaml(homeDir string) []byte {
	var (
		b   []byte
		err error
	)

	if _, err = os.Stat(path.Join(homeDir, componentsYaml)); err == nil {
		logrus.Debugf("load components from %s", path.Join(homeDir, componentsYaml))
		fsys := manifests.BuiltinOrDir(homeDir)
		b, err = fs.ReadFile(fsys, "components.yaml")
	}

	if err != nil {
		// fail back to built-in configuration
		fsys := manifests.BuiltinOrDir("")
		b, err = fs.ReadFile(fsys, builtinComponentsYaml)
		if err != nil {
			logrus.Fatalf("failed ummarshal components: %v", err)
		}
	}

	return b
}

func (g *Options) ReloadComponents() {
	g.Components = loadComponents(g.HomeDir)
}

func (g *Options) Errorf(format string, a ...interface{}) {
	if g.Ui == nil {
		return
	}
	g.Ui.Error(fmt.Sprintf(format, a...))
}
