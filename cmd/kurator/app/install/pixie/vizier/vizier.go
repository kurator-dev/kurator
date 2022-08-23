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

package vizier

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"kurator.dev/kurator/pkg/generic"
	plugin "kurator.dev/kurator/pkg/plugin/pixie/vizier"
)

const (
	communityCloudAddr = "withpixie.ai:443"
	pxNamespace        = "px"
)

func NewCmd(opts *generic.Options) *cobra.Command {
	pluginArgs := plugin.InstallArgs{}
	vizierCmd := &cobra.Command{
		Use:   "vizier",
		Short: "Install vizier component",
		RunE: func(c *cobra.Command, args []string) error {
			plugin, err := plugin.NewPlugin(opts, &pluginArgs)
			if err != nil {
				logrus.Errorf("pixie vizier init error: %v", err)
				return fmt.Errorf("pixie vizier init error: %v", err)
			}

			if err := plugin.Execute(args, nil); err != nil {
				logrus.Errorf("pixie vizier execute error: %v", err)
				return fmt.Errorf("pixie vizier execute error: %v", err)
			}
			logrus.Info("pixie vizier install completed.")
			return nil
		},
	}

	vizierCmd.PersistentFlags().StringVar(&pluginArgs.PxNamespace, "px-namespace", pxNamespace, "The namespace use to install vizier.")
	vizierCmd.PersistentFlags().StringVar(&pluginArgs.CloudAddress, "cloud-addr", communityCloudAddr, "The address of the Pixie cloud instance that the vizier should be connected to.")
	vizierCmd.PersistentFlags().StringVar(&pluginArgs.DeployKey, "deploy-key", "", "The deploy key is used to link the deployed vizier to a specific user/project.")

	return vizierCmd
}
