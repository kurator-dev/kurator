package kubeedge

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/zirain/ubrain/pkg/generic"
	plugin "github.com/zirain/ubrain/pkg/plugin/kubeedge"
)

var pluginArgs = plugin.InstallArgs{}

func NewCmd(opts *generic.Options) *cobra.Command {
	kubeedgeCmd := &cobra.Command{
		Use:   "kubeedge",
		Short: "Install kubeedge component",
		RunE: func(c *cobra.Command, args []string) error {
			if len(pluginArgs.Clusters) == 0 {
				logrus.Errorf("Please provide at least 1 cluster")
				return fmt.Errorf("must provide at least 1 cluster")
			}

			plugin, err := plugin.NewKubeEdgePlugin(opts, &pluginArgs)
			if err != nil {
				logrus.Errorf("kubeedge init error: %v", err)
				return fmt.Errorf("kubeedge init error: %v", err)
			}

			logrus.Debugf("start install kubeedge Global: %+v ", opts)
			if err := plugin.Execute(args, nil); err != nil {
				logrus.Errorf("kubeedge execute error: %v", err)
				return fmt.Errorf("kubeedge execute error: %v", err)
			}

			return nil
		},
	}

	kubeedgeCmd.PersistentFlags().StringSliceVar(&pluginArgs.Clusters, "cluster", nil, "Comma separated list of clusters to install KubeEdge.")
	kubeedgeCmd.PersistentFlags().StringVarP(&pluginArgs.Namespace, "namespace", "n", "kubeedge", "The Namespace to install KubeEdge, default value is 'kubeedge'.")
	kubeedgeCmd.PersistentFlags().StringVar(&pluginArgs.AdvertiseAddress, "advertise-address", "", "Use this key to set IPs in cloudcore's certificate SubAltNames field. eg: 10.10.102.78,10.10.102.79")

	return kubeedgeCmd
}
