package karmada

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/plugin/join"
)

var pluginArgs = join.Args{}

func NewCmd(opts *generic.Options) *cobra.Command {
	joinCmd := &cobra.Command{
		Use:                   "karmada [cluster]",
		Short:                 "Registers a cluster to karmada.",
		DisableFlagsInUseLine: true,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				logrus.Errorf("cluster name is required")
				return errors.New("cluster name is required")
			}
			pluginArgs.ClusterName = args[0]

			plugin, err := join.NewJoinPlugin(opts, &pluginArgs)
			if err != nil {
				logrus.Errorf("join init error: %v", err)
				return fmt.Errorf("join init error: %v", err)
			}

			logrus.Debugf("start join cluster %s", pluginArgs.ClusterName)
			if err := plugin.Execute(args, nil); err != nil {
				logrus.Errorf("join execute error: %v", err)
				return fmt.Errorf("join execute error: %v", err)
			}

			return nil
		},
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}

	joinCmd.PersistentFlags().StringVar(&pluginArgs.ClusterKubeConfig, "cluster-kubeconfig", "", "Path of the cluster's kubeconfig.")
	joinCmd.PersistentFlags().StringVar(&pluginArgs.ClusterContext, "cluster-context", "",
		"Context name of cluster in kubeconfig. Only works when there are multiple contexts in the kubeconfig.")

	return joinCmd
}
