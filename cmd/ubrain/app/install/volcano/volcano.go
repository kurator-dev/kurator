package volcano

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/zirain/ubrain/pkg/generic"
	plugin "github.com/zirain/ubrain/pkg/plugin/volcano"
)

var pluginArgs = plugin.InstallArgs{}

func NewCmd(opts *generic.Options) *cobra.Command {
	volcanoCmd := &cobra.Command{
		Use:   "volcano",
		Short: "Install volcano component",
		RunE: func(c *cobra.Command, args []string) error {
			plugin, err := plugin.NewPlugin(opts, &pluginArgs)
			if err != nil {
				logrus.Errorf("volcano init error: %v", err)
				return fmt.Errorf("volcano init error: %v", err)
			}

			logrus.Debugf("start install volcano Global: %+v ", opts)
			if err := plugin.Execute(args, nil); err != nil {
				logrus.Errorf("volcano execute error: %v", err)
				return fmt.Errorf("volcano execute error: %v", err)
			}

			return nil
		},
	}

	volcanoCmd.PersistentFlags().StringSliceVar(&pluginArgs.Clusters, "cluster", nil, "Comma separated list of clusters to install volcano.")

	return volcanoCmd
}
