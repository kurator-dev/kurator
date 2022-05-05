package command

import (
	"strings"

	flag "github.com/spf13/pflag"
	"istio.io/istio/pkg/url"

	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/plugin/istio"
)

type IstioInstallCommand struct {
	*generic.Options
}

func (c *IstioInstallCommand) Run(args []string) int {
	istioCfg, ok := c.Components["istio"]
	if !ok {
		c.Ui.Error("Failed to load istio component")
		return 1
	}

	istioArgs := &istio.InstallArgs{
		Hub: istioCfg.Hub,
		Tag: istioCfg.Version,
	}
	cmdFlags := c.istioFlagSet(c.Options, istioArgs)
	cmdFlags.Usage = func() {
		c.Errorf(c.Help())
		c.Errorf(cmdFlags.FlagUsages())
	}
	if err := cmdFlags.Parse(args); err != nil {
		c.Errorf("Error parsing command-line flags: %s\n", err.Error())
		return 1
	}

	plugin, err := istio.NewIstioPluginHandler(c.Options, istioArgs)
	if err != nil {
		c.Errorf("istio init error: %v", err)
		return 1
	}

	c.Infof("start install istio Global: %+v ", c.Options)
	if err := plugin.Execute(args, nil); err != nil {
		c.Infof("istio execute error: %v", err)
		return 1
	}

	return 0
}

func (c *IstioInstallCommand) Help() string {
	helpText := `
Usage: ubrain install istio [Options]
`
	return strings.TrimSpace(helpText)
}

func (c *IstioInstallCommand) Synopsis() string {
	return "Install istio component"
}

func (c *IstioInstallCommand) istioFlagSet(s *generic.Options, args *istio.InstallArgs) *flag.FlagSet {
	f := c.FlagSet("istio")

	s.AddFlags(f)

	// TODO: add comments about the flag usage link
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
