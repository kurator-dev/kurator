/*
Copyright 2022-2025 Kurator Authors.

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

package kubeedge

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"kurator.dev/kurator/pkg/generic"
	plugin "kurator.dev/kurator/pkg/plugin/kubeedge"
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
