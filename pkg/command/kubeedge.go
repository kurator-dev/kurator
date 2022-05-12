package command

import (
	"strings"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/plugin/kubeedge"
)

type KubeEdgeInstallCommand struct {
	*generic.Options
}

func (c *KubeEdgeInstallCommand) Run(args []string) int {
	edgeArgs := &kubeedge.InstallArgs{}
	cmdFlags := c.kubeedgeFlagSet(c.Options, edgeArgs)
	cmdFlags.Usage = func() {
		c.Errorf(c.Help())
		c.Errorf(cmdFlags.FlagUsages())
	}

	if err := cmdFlags.Parse(args); err != nil {
		logrus.Errorf("Error parsing command-line flags: %s", err.Error())
		return 1
	}

	if len(edgeArgs.Clusters) == 0 {
		logrus.Errorf("Please provider at least 1 cluster")
		return 1
	}

	plugin, err := kubeedge.NewKubeEdgePlugin(c.Options, edgeArgs)
	if err != nil {
		logrus.Errorf("kubeedge init error: %v", err)
		return 1
	}

	logrus.Debugf("start install kubeedge: %+v ", c.Options)
	if err := plugin.Execute(args, nil); err != nil {
		logrus.Errorf("kubeedge execute error: %v", err)
		return 1
	}

	return 0
}

func (c *KubeEdgeInstallCommand) Help() string {
	helpText := `
Usage: ubrain install kubeedge [Options]
`
	return strings.TrimSpace(helpText)
}

func (c *KubeEdgeInstallCommand) Synopsis() string {
	return "Install kubeedge component"
}

func (c *KubeEdgeInstallCommand) kubeedgeFlagSet(s *generic.Options, args *kubeedge.InstallArgs) *flag.FlagSet {
	f := c.FlagSet("kubeedge")
	s.AddFlags(f)

	// TODO: add comments about the flag usage link
	f.StringSliceVar(&args.Clusters, "cluster", nil, "Comma separated list of clusters to install KubeEdge.")
	f.StringVarP(&args.Namespace, "namespace", "n", "kubeedge", "The Namespace to install KubeEdge, default value is 'kubeedge'.")
	return f
}
