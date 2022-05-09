package generic

import (
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/mitchellh/cli"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/yaml"

	"github.com/zirain/ubrain/manifests"
)

type Options struct {
	DryRun bool // TODO: support dry run

	config *genericclioptions.ConfigFlags

	Ui cli.Ui
	// HomeDir is an absolute path which most importantly contains "versions" installed from binary. Defaults to DefaultHomeDir
	HomeDir string
	TempDir string

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
	g.Components = loadComponents()
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

	tempDir, err := ioutil.TempDir(os.TempDir(), "ubrain")
	if err != nil {
		tempDir = os.TempDir()
	}
	fs.StringVar(&g.HomeDir, "home-dir", path.Join(homeDir, ".ubrain"), "install path, default to $HOME/.ubrain")
	fs.StringVar(&g.TempDir, "temp-dir", tempDir, "file path including temporary generated files")

	// use karmada apiserver by default
	fs.StringVarP(&g.KubeConfig, "kubeconfig", "c", path.Join(homeDir, ".kube/karmada.config"), "path to the kubeconfig file")
	fs.StringVar(&g.KubeContext, "context", "karmada-apiserver", "name of the kubeconfig context to use")

	fs.BoolVar(&g.DryRun, "dry-run", false, "console/log output only, make no changes.")
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

func loadComponents() map[string]Component {
	fsys := manifests.BuiltinOrDir("")
	b, err := fs.ReadFile(fsys, "profiles/components.yaml")
	if err != nil {
		log.Printf("failed ummarshal components: %v\n", err)
	}
	var c cfg
	if err := yaml.Unmarshal(b, &c); err != nil {
		log.Printf("failed ummarshal components: %v\n", err)
	}

	components := make(map[string]Component, len(c.Components))
	for _, com := range c.Components {
		components[com.Name] = com
	}

	return components
}
