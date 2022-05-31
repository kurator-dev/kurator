package install

import (
	"github.com/spf13/cobra"

	"kurator.dev/kurator/cmd/kurator/app/install/istio"
	"kurator.dev/kurator/cmd/kurator/app/install/karmada"
	"kurator.dev/kurator/cmd/kurator/app/install/kubeedge"
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
	return installCmd
}
