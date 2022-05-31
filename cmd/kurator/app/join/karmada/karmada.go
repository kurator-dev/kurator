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
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/plugin/join"
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
