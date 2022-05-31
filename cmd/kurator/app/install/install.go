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

package install

import (
	"github.com/spf13/cobra"

	"kurator.dev/kurator/cmd/kurator/app/install/istio"
	"kurator.dev/kurator/cmd/kurator/app/install/karmada"
	"kurator.dev/kurator/cmd/kurator/app/install/kubeedge"
	"kurator.dev/kurator/cmd/kurator/app/install/prometheus"
	"kurator.dev/kurator/cmd/kurator/app/install/volcano"
	"kurator.dev/kurator/pkg/generic"
)

func NewCmd(opts *generic.Options) *cobra.Command {
	installCmd := &cobra.Command{
		Use:                   "install",
		Short:                 "install target component",
		DisableFlagsInUseLine: true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}

	installCmd.AddCommand(istio.NewCmd(opts))
	installCmd.AddCommand(karmada.NewCmd(opts))
	installCmd.AddCommand(kubeedge.NewCmd(opts))
	installCmd.AddCommand(volcano.NewCmd(opts))
	installCmd.AddCommand(prometheus.NewCmd(opts))
	return installCmd
}
