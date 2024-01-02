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

package pipeline

import (
	"github.com/spf13/cobra"

	"kurator.dev/kurator/cmd/kurator/app/pipeline/execution"
	"kurator.dev/kurator/pkg/generic"
)

func NewCmd(opts *generic.Options) *cobra.Command {
	joinCmd := &cobra.Command{
		Use:                   "pipeline",
		Short:                 "manage kurator pipeline",
		DisableFlagsInUseLine: true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
	}

	joinCmd.AddCommand(execution.NewCmd(opts))

	return joinCmd
}
