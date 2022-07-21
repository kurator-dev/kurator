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

package istio

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"istio.io/istio/pkg/url"

	"kurator.dev/kurator/pkg/generic"
	plugin "kurator.dev/kurator/pkg/plugin/istio"
)

var pluginArgs = plugin.InstallArgs{}

var supportedNetworkModes = map[string]struct{}{
	"flat":     {},
	"non-flat": {},
}

func NewCmd(opts *generic.Options) *cobra.Command {
	istioCmd := &cobra.Command{
		Use:   "istio",
		Short: "Install istio component",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if _, ok := supportedNetworkModes[pluginArgs.NetworkMode]; !ok {
				return fmt.Errorf("%s network mode is not supported", pluginArgs.NetworkMode)
			}
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			istioCfg, ok := opts.Components["istio"]
			if !ok {
				logrus.Errorf("Failed to load istio component")
				return errors.New("failed to load istio component")
			}

			pluginArgs.Hub = istioCfg.Hub
			pluginArgs.Tag = istioCfg.Version

			plugin, err := plugin.NewIstioPlugin(opts, &pluginArgs)
			if err != nil {
				logrus.Errorf("istio init error: %v", err)
				return fmt.Errorf("istio init error: %v", err)
			}

			logrus.Debugf("start install istio Global: %+v ", opts)
			if err := plugin.Execute(args, nil); err != nil {
				logrus.Errorf("istio execute error: %v", err)
				return fmt.Errorf("istio execute error: %v", err)
			}

			return nil
		},
	}

	f := istioCmd.PersistentFlags()
	f.StringVar(&pluginArgs.Primary, "primary", "", "The cluster name of the istio control plane.")
	f.StringSliceVar(&pluginArgs.Remotes, "remote", nil, "The name of the istio remote cluster.")
	f.StringVar(&pluginArgs.NetworkMode, "network-mode", "flat", "The network of the istio remote cluster, support flat/non-flat mode, default value is falt.")

	f.StringSliceVarP(&pluginArgs.IopFiles, "filename", "f", nil, `Path to file containing IstioOperator custom resource 
	This flag can be specified multiple times to overlay multiple files. Multiple files are overlaid in left to right order.`)

	f.StringArrayVarP(&pluginArgs.SetFlags, "set", "s", nil, `Override an IstioOperator value, e.g. to choose a profile
	(--set profile=demo), enable or disable components (--set components.cni.enabled=true), or override Istio
	settings (--set meshConfig.enableTracing=true). See documentation for more info:`+url.IstioOperatorSpec)

	f.StringVar(&pluginArgs.Cacerts, "cacert", "", "The root cacerts of the istio, self-signed certs will be used if empty.")

	return istioCmd
}
