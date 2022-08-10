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

package argocd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"kurator.dev/kurator/pkg/generic"
	plugin "kurator.dev/kurator/pkg/plugin/argocd"
)

var pluginArgs = plugin.InstallArgs{}

func NewCmd(opts *generic.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "argocd",
		Short: "Install argocd component",
		RunE: func(c *cobra.Command, args []string) error {
			plugin, err := plugin.NewArgoCDPlugin(opts, &pluginArgs)
			if err != nil {
				logrus.Errorf("argocd plugin init error: %v", err)
				return err
			}

			logrus.Debugf("start install argoCD: %+v ", opts)
			if err := plugin.Execute(args, nil); err != nil {
				logrus.Errorf("argoCD execute error: %v", err)
				return err
			}

			return nil
		},
	}

	f := cmd.PersistentFlags()
	f.StringVar(&pluginArgs.ClusterKubeconfig, "cluster-kubeconfig", "/etc/karmada/karmada-apiserver.config",
		"Karmada apiserver kubeconfig, default to /etc/karmada/karmada-apiserver.config")
	f.StringVar(&pluginArgs.ClusterContext, "cluster-context", "karmada", "name of karmada context")

	return cmd
}
