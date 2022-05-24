package app

import (
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/zirain/ubrain/cmd/ubrain/app/install"
	"github.com/zirain/ubrain/cmd/ubrain/app/join"
	"github.com/zirain/ubrain/cmd/ubrain/app/version"
	"github.com/zirain/ubrain/pkg/generic"
)

func Run() error {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	c := NewUbarinCommand()
	return c.Execute()
}

// NewUbarinCommand returns a cobra.Command to run ubrain commands
func NewUbarinCommand() *cobra.Command {
	o := generic.New()
	ctl := &cobra.Command{
		Use: "ubrain", // TODO: rename and add project description?
	}

	ctl.ResetFlags()
	o.AddFlags(ctl.PersistentFlags())

	ctl.AddCommand(version.NewCmd())
	ctl.AddCommand(install.NewCmd(o))
	ctl.AddCommand(join.NewCmd(o))

	return ctl
}
