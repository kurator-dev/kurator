/*
Copyright 2022-2025 Kurator Authors.

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

package app

import (
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"kurator.dev/kurator/cmd/kurator/app/install"
	"kurator.dev/kurator/cmd/kurator/app/join"
	"kurator.dev/kurator/cmd/kurator/app/pipeline"
	"kurator.dev/kurator/cmd/kurator/app/tool"
	"kurator.dev/kurator/cmd/kurator/app/version"
	"kurator.dev/kurator/pkg/generic"
)

func Run() error {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	c := NewKuratorCommand()
	return c.Execute()
}

// NewKuratorCommand returns a cobra.Command to run kurator commands
func NewKuratorCommand() *cobra.Command {
	o := generic.New()
	cmd := &cobra.Command{
		Use:          "kurator",
		Short:        "Kurator builds distributed cloud-native stacks.",
		SilenceUsage: true,
	}

	cmd.ResetFlags()
	o.AddFlags(cmd.PersistentFlags())
	o.ReloadComponents()

	cmd.AddCommand(version.NewCmd())
	cmd.AddCommand(install.NewCmd(o))
	cmd.AddCommand(join.NewCmd(o))
	cmd.AddCommand(tool.NewCmd(o))
	cmd.AddCommand(pipeline.NewCmd(o))

	return cmd
}
