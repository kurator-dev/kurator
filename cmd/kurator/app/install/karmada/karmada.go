/*
Copyright Kurator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package karmada

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"kurator.dev/kurator/pkg/generic"
	plugin "kurator.dev/kurator/pkg/plugin/karmada"
)

var pluginArgs = plugin.InstallArgs{}

func NewCmd(opts *generic.Options) *cobra.Command {
	karmadaCmd := &cobra.Command{
		Use:   "karmada",
		Short: "Install karmada component",
		RunE: func(c *cobra.Command, args []string) error {
			plugin, err := plugin.NewKarmadaPlugin(opts, &pluginArgs)
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

	f := karmadaCmd.PersistentFlags()
	f.StringArrayVar(&pluginArgs.SetFlags, "set", nil, "set karmada install parameters, e.g. --set karmada-data=/etc/karmada")

	return karmadaCmd
}
