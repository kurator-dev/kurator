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

package volcano

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"kurator.dev/kurator/pkg/generic"
	plugin "kurator.dev/kurator/pkg/plugin/volcano"
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
