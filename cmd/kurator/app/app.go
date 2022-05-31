package app

import (
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"kurator.dev/kurator/cmd/kurator/app/install"
	"kurator.dev/kurator/cmd/kurator/app/join"
	"kurator.dev/kurator/cmd/kurator/app/version"
	"kurator.dev/kurator/pkg/generic"
)

func Run() error {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	c := NewUbarinCommand()
	return c.Execute()
}

// NewUbarinCommand returns a cobra.Command to run kurator commands
func NewUbarinCommand() *cobra.Command {
	o := generic.New()
	ctl := &cobra.Command{
		Use: "kurator", // TODO: rename and add project description?
	}

	ctl.ResetFlags()
	o.AddFlags(ctl.PersistentFlags())

	ctl.AddCommand(version.NewCmd())
	ctl.AddCommand(install.NewCmd(o))
	ctl.AddCommand(join.NewCmd(o))

	return ctl
}
