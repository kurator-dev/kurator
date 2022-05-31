package karmada

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"kurator.dev/kurator/pkg/generic"
	plugin "kurator.dev/kurator/pkg/plugin/karmada"
)

func NewCmd(opts *generic.Options) *cobra.Command {
	karmadaCmd := &cobra.Command{
		Use:   "karmada",
		Short: "Install karmada component",
		RunE: func(c *cobra.Command, args []string) error {
			plugin, err := plugin.NewKarmadaPlugin(opts)
			if err != nil {
				logrus.Errorf("karmada init error: %v", err)
				return fmt.Errorf("karmada init error: %v", err)
			}

			logrus.Debugf("start install karmada Global: %+v ", opts)
			if err := plugin.Execute(args, nil); err != nil {
				logrus.Errorf("karmada execute error: %v", err)
				return fmt.Errorf("karmada execute error: %v", err)
			}

			return nil
		},
	}

	return karmadaCmd
}
