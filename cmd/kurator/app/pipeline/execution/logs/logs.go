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

package logs

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/pipeline/execution/logs"
)

func NewCmd(opts *generic.Options) *cobra.Command {
	var Args = logs.Args{}
	logsCmd := &cobra.Command{
		Use:     "logs",
		Short:   "Display aggregated logs from multiple tasks within kurator pipeline execution",
		Example: getExample(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("please specify the pipeline execution name")
			}

			pipelineExecutionName := args[0] // the first one in args[] is the name of pipeline execution

			PipelineList, err := logs.NewPipelineLogs(opts, &Args, pipelineExecutionName)
			if err != nil {
				logrus.Errorf("pipeline excution logs init error: %v", err)
				return fmt.Errorf("pipeline excution logs init error: %v", err)
			}

			logrus.Debugf("start logs pipeline execution obj, Global: %+v ", opts)
			if err := PipelineList.LogsExecute(); err != nil {
				logrus.Errorf("pipeline logs execute error: %v", err)
				return fmt.Errorf("pipeline logs execute error: %v", err)
			}

			return nil
		},
	}

	logsCmd.PersistentFlags().StringVarP(&Args.Namespace, "namespace", "n", "default", "specific namespace")
	logsCmd.PersistentFlags().Int64Var(&Args.TailLines, "tail", 0, "number of lines to display from the end of the logs in each task pod container, must be greater than 0 to take effect")

	return logsCmd
}

func getExample() string {
	return `  # Display aggregated logs from an example pipeline execution in the default namespace
  kurator pipeline execution logs example-pipeline-execution

  # Display aggregated logs from an example pipeline execution in a specific namespace (replace 'example-namespace' with your namespace)
  kurator pipeline execution logs example-pipeline-execution -n example-namespace

  # Display the last 10 lines of logs from an example pipeline execution
  kurator pipeline execution logs example-pipeline-execution --tail 10
`
}
