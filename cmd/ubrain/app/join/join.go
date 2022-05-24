package join

import (
	"github.com/spf13/cobra"

	"github.com/zirain/ubrain/cmd/ubrain/app/join/karmada"
	"github.com/zirain/ubrain/pkg/generic"
)

func NewCmd(opts *generic.Options) *cobra.Command {
	joinCmd := &cobra.Command{
		Use:                   "join",
		Short:                 "Register a cluster or node",
		DisableFlagsInUseLine: true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}

	joinCmd.AddCommand(karmada.NewCmd(opts))

	return joinCmd
}
