package command

import (
	"strings"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/plugin/join"
)

type JoinCommand struct {
	*generic.Options
}

func (c *JoinCommand) Run(args []string) int {
	joinArgs := &join.Args{}
	cmdFlags := c.flagSet(c.Options, joinArgs)
	cmdFlags.Usage = func() {
		c.Errorf(c.Help())
		c.Errorf(cmdFlags.FlagUsages())
	}
	if len(args) == 0 {
		logrus.Errorf("cluster name is required")
		return 1
	}
	joinArgs.ClusterName = args[0]
	if err := cmdFlags.Parse(args); err != nil {
		logrus.Errorf("Error parsing command-line flags: %s", err.Error())
		return 1
	}

	plugin, err := join.NewJoinPlugin(c.Options, joinArgs)
	if err != nil {
		logrus.Errorf("join init error: %v", err)
		return 1
	}

	logrus.Infof("start join cluster %s", joinArgs.ClusterName)
	if err := plugin.Execute(args, nil); err != nil {
		logrus.Infof("join execute error: %v", err)
		return 1
	}

	return 0
}

func (c *JoinCommand) Help() string {
	helpText := `
Usage: ubrain join CLUSTER_NAME --cluster-kubeconfig=<KUBECONFIG> --cluster-context=<CTX>
`
	return strings.TrimSpace(helpText)
}

func (c *JoinCommand) Synopsis() string {
	return "Join registers a cluster to karmada."
}

func (c *JoinCommand) flagSet(s *generic.Options, args *join.Args) *flag.FlagSet {
	f := c.FlagSet("join")
	s.AddFlags(f)
	f.StringVar(&args.ClusterKubeConfig, "cluster-kubeconfig", "", "Path of the cluster's kubeconfig.")
	f.StringVar(&args.ClusterContext, "cluster-context", "",
		"Context name of cluster in kubeconfig. Only works when there are multiple contexts in the kubeconfig.")
	return f
}
