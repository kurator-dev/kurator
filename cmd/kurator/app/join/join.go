package join

import (
	"github.com/spf13/cobra"

	"kurator.dev/kurator/cmd/kurator/app/join/karmada"
	"kurator.dev/kurator/cmd/kurator/app/join/kubeedge"
	"kurator.dev/kurator/pkg/generic"
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
	joinCmd.AddCommand(kubeedge.NewCmd(opts))

	return joinCmd
}
