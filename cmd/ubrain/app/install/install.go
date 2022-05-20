package install

import (
	"github.com/spf13/cobra"

	"github.com/zirain/ubrain/cmd/ubrain/app/install/istio"
	"github.com/zirain/ubrain/cmd/ubrain/app/install/karmada"
	"github.com/zirain/ubrain/cmd/ubrain/app/install/kubeedge"
	"github.com/zirain/ubrain/cmd/ubrain/app/install/volcano"
	"github.com/zirain/ubrain/pkg/generic"
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
