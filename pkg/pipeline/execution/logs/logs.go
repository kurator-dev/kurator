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
	"bytes"
	"context"
	"io"

	"github.com/sirupsen/logrus"
	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
)

// pipelineLogs is used to handle and display aggregated logs from a specific pipeline execution.
type pipelineLogs struct {
	*client.Client       // Embedded client for API interactions.
	args           *Args // Arguments passed through command line flags.
	options        *generic.Options
	name           string // Name of the pipeline execution to fetch logs for.
}

// Args holds the command line arguments for the logs command.
type Args struct {
	Namespace string // Namespace from which to fetch the logs.
	TailLines int64  // Number of lines to display from the end of the logs in each task pod container, must be greater than 0 to take effect
}

// NewPipelineLogs creates a new pipelineLogs instance.
func NewPipelineLogs(opts *generic.Options, args *Args, pipelineExecutionName string) (*pipelineLogs, error) {
	pList := &pipelineLogs{
		options: opts,
		args:    args,
		name:    pipelineExecutionName,
	}
	rest := opts.RESTClientGetter()
	c, err := client.NewClient(rest)
	if err != nil {
		return nil, err
	}
	pList.Client = c
	return pList, nil
}

// LogsExecute fetches and displays aggregated logs from pipeline execution.
func (p *pipelineLogs) LogsExecute() error {
	namespace := p.args.Namespace

	pipelineRun := &tektonapi.PipelineRun{}

	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      p.name,
	}

	// Retrieve the PipelineRun object
	if err := p.CtrlRuntimeClient().Get(context.Background(), namespacedName, pipelineRun); err != nil {
		logrus.Errorf("failed to get PipelineRun '%s' in namespace '%s', %v", p.name, namespace, err)
		return err
	}

	// Retrieve the ChildReferences of TaskRun
	taskPodList := pipelineRun.Status.ChildReferences

	// Iterate through each TaskRun to fetch their logs
	for _, taskRef := range taskPodList {
		if taskRef.Kind == "TaskRun" {
			logrus.Infof("Fetching logs for TaskRun: %s", taskRef.Name)
			if err := p.fetchAndPrintPodLogs(taskRef.Name, namespace); err != nil {
				logrus.Errorf("failed to fetch logs for TaskRun '%s' in namespace '%s', %v", taskRef.Name, namespace, err)
				// here just continue to attempt fetching logs for other TaskRuns, rather than directly return an error.
			}
		}
	}

	return nil
}

// fetchAndPrintPodLogs fetches and prints the logs of the pod associated with a given TaskRun.
func (p *pipelineLogs) fetchAndPrintPodLogs(taskRunName, namespace string) error {
	podName := getPodNameFromTaskRun(taskRunName)
	pod := &corev1.Pod{}

	// Get the details of the Pod
	if err := p.Client.CtrlRuntimeClient().Get(context.Background(), ctrlclient.ObjectKey{Name: podName, Namespace: namespace}, pod); err != nil {
		return err
	}

	// Iterate through each container in the Pod to fetch logs
	for _, container := range pod.Spec.Containers {
		logrus.Infof("Fetching logs for container '%s' in Pod '%s'", container.Name, podName)

		podLogOpts := corev1.PodLogOptions{Container: container.Name}
		if p.args.TailLines > 0 {
			podLogOpts.TailLines = &p.args.TailLines
		}

		req := p.Client.KubeClient().CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)

		podLogs, err := req.Stream(context.Background())
		if err != nil {
			logrus.Errorf("Failed to fetch logs for container '%s', %v", container.Name, err)
			continue
		}

		// Read and display the logs
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		podLogs.Close()
		if err != nil {
			logrus.Errorf("Failed to read logs for container '%s', %v", container.Name, err)
			continue
		}

		logrus.Infof("Logs from container '%s':\n%s", container.Name, buf.String())
	}

	return nil
}

// getPodNameFromTaskRun gets the pod name for a taskrun. The taskRunName and the name of the pod executing the task differ only by "-pod"
func getPodNameFromTaskRun(taskRunName string) string {
	return taskRunName + "-pod"
}
