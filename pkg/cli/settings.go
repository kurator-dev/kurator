package cli

import (
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"

	"github.com/mitchellh/cli"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/zirain/ubrain/manifests"
)

func New() *Settings {
	g := &Settings{
		Ui: &cli.BasicUi{
			Writer:      os.Stdout,
			ErrorWriter: os.Stdout,
			Reader:      os.Stdin,
		},
		RootFS: manifests.BuiltinOrDir(""),
	}
	g.Components = loadComponets(g.RootFS)
	// bind to kubernetes config flags
	g.config = &genericclioptions.ConfigFlags{
		Context:    &g.KubeContext,
		KubeConfig: &g.KubeConfig,
	}
	return g
}

type Settings struct {
	DryRun bool // TODO: support dry run

	config *genericclioptions.ConfigFlags
	RootFS fs.FS

	Ui cli.Ui
	// HomeDir is an absolute path which most importantly contains "versions" installed from binary. Defaults to DefaultHomeDir
	HomeDir string
	TempDir string

	KubeConfig  string
	KubeContext string

	Components map[string]Component
}

// AddFlags binds flags to the given flagset.
func (s *Settings) AddFlags(fs *pflag.FlagSet) {
	var homeDir, tempDir string
	if s, err := user.Current(); err == nil {
		homeDir = s.HomeDir
	} else {
		homeDir = os.TempDir()
	}

	tempDir, err := ioutil.TempDir(os.TempDir(), "ubrain")
	if err != nil {
		tempDir = os.TempDir()
	}
	fs.StringVarP(&s.HomeDir, "home-dir", "", path.Join(homeDir, ".ubrain"), "path to the kubeconfig file")
	fs.StringVar(&s.TempDir, "temp-dir", tempDir, "name of the kubeconfig context to use")

	// use karmada apiserver by default
	fs.StringVarP(&s.KubeConfig, "kubeconfig", "c", path.Join(homeDir, ".kube/karmada.config"), "path to the kubeconfig file")
	fs.StringVar(&s.KubeContext, "context", "karmada-apiserver", "name of the kubeconfig context to use")

	fs.BoolVar(&s.DryRun, "dry-run", false, "console/log output only, make no changes.")
}

// RESTClientGetter gets the kubeconfig from EnvSettings
func (s *Settings) RESTClientGetter() genericclioptions.RESTClientGetter {
	return s.config
}

type Component struct {
	Name    string
	Version string
}

func loadComponets(files fs.FS) map[string]Component {
	type cfg struct {
		Components []Component `json:"components"`
	}
	var c cfg
	b, err := fs.ReadFile(files, "profiles/components.yaml")
	if err != nil {
		log.Printf("failed ummarshal components: %v\n", err)
	}

	if err := yaml.Unmarshal(b, &c); err != nil {
		log.Printf("failed ummarshal components: %v\n", err)
	}

	components := make(map[string]Component)

	for _, com := range c.Components {
		components[com.Name] = com
	}

	return components
}
