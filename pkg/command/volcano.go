package command

import (
	"strings"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/plugin/volcano"
)

type VolcanoInstallCommand struct {
	*generic.Options
}

func (c *VolcanoInstallCommand) Run(args []string) int {
	_, ok := c.Components["volcano"]
	if !ok {
		logrus.Errorf("Failed to load volcano component")
		return 1
	}

	volcanoArgs := &volcano.InstallArgs{}
	cmdFlags := c.volcanoFlagSet(c.Options, volcanoArgs)
	cmdFlags.Usage = func() {
		c.Errorf(c.Help())
		c.Errorf(cmdFlags.FlagUsages())
	}
	if err := cmdFlags.Parse(args); err != nil {
		logrus.Errorf("Error parsing command-line flags: %s", err.Error())
		return 1
	}

	if len(volcanoArgs.Clusters) == 0 {
		logrus.Errorf("Please provider at least 1 cluster")
		return 1
	}

	plugin, err := volcano.NewPlugin(c.Options, volcanoArgs)
	if err != nil {
		logrus.Errorf("volcano init error: %v", err)
		return 1
	}

	if err := plugin.Execute(args, nil); err != nil {
		logrus.Errorf("volcano execute error: %v", err)
		return 1
	}

	return 0
}

func (c *VolcanoInstallCommand) Help() string {
	helpText := `
Usage: ubrain install volcano [Options]
`
	return strings.TrimSpace(helpText)
}

func (c *VolcanoInstallCommand) Synopsis() string {
	return "Install volcano component"
}

func (c *VolcanoInstallCommand) volcanoFlagSet(s *generic.Options, args *volcano.InstallArgs) *flag.FlagSet {
	f := c.FlagSet("volcano")

	s.AddFlags(f)

	// TODO: add comments about the flag usage link
	f.StringSliceVar(&args.Clusters, "cluster", nil, "Comma separated list of clusters to install volcano.")
	return f
}
