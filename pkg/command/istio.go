package command

import (
	"strings"

	flag "github.com/spf13/pflag"
	"istio.io/istio/pkg/url"

	"github.com/zirain/ubrain/pkg/cli"
	"github.com/zirain/ubrain/pkg/plugin/istio"
)

type IstioInstallCommad struct {
	Base
}

func (c *IstioInstallCommad) Run(args []string) int {
	if _, ok := c.Settings.Components["istio"]; !ok {
		c.Settings.Ui.Error("Failed to load istio component")
		return 1
	}

	istioArgs := &istio.InstallArgs{
		Hub: "docker.io/istio", //TODO: make this configurable
		Tag: c.Settings.Components["istio"].Version,
	}
	cmdFlags := c.istioFlagSet(c.Settings, istioArgs)
	cmdFlags.Usage = func() { c.Settings.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Errorf("Error parsing command-line flags: %s\n", err.Error())
		return 1
	}

	plugin, err := istio.NewIstioPluginHandler(c.Settings, istioArgs)
	if err != nil {
		c.Infof("istio init error: %v", err)
		return 1
	}

	c.Infof("start install istio Global: %+v ", c.Base.Settings)
	if err := plugin.Execute(args, nil); err != nil {
		c.Infof("istio execute error: %v", err)
		return 1
	}

	return 0
}

func (c *IstioInstallCommad) Help() string {
	helpText := `
Usage: ubrain install istio [options]
`
	return strings.TrimSpace(helpText)
}

func (c *IstioInstallCommad) Synopsis() string {
	return "Install istio component"
}

func (c *IstioInstallCommad) istioFlagSet(s *cli.Settings, args *istio.InstallArgs) *flag.FlagSet {
	f := c.Base.FlagSet("istio")

	s.AddFlags(f)

	f.StringSliceVarP(&args.IopFiles, "filename", "f", nil, `Path to file containing IstioOperator custom resource
	This flag can be specified multiple times to overlay multiple files. Multiple files are overlaid in left to right order.`)
	f.StringArrayVarP(&args.SetFlags, "set", "s", nil, `Override an IstioOperator value, e.g. to choose a profile
	(--set profile=demo), enable or disable components (--set components.cni.enabled=true), or override Istio
	settings (--set meshConfig.enableTracing=true). See documentation for more info:`+url.IstioOperatorSpec)
	f.StringVar(&args.Primary, "primary", "member1", "The cluster name of the istio control plane.")
	f.StringSliceVar(&args.Remotes, "remote", []string{"member2"}, "The name of the istio remote cluster.")
	f.StringVar(&args.Cacerts, "cacert", "/root/istio-certs/primary", "The root cacerts of the istio.")
	return f
}
