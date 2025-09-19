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

package list

import (
	"context"
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"
	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
)

// pipelineList is the structure used for listing pipeline objects.
type pipelineList struct {
	*client.Client
	args    *Args
	options *generic.Options
}

// Args holds the arguments for listing pipeline runs.
type Args struct {
	Namespace     string // Specific namespace to list pipeline runs.
	AllNamespaces bool   // Flag to list pipeline runs across all namespaces.
}

// NewPipelineList creates a new pipelineList instance.
func NewPipelineList(opts *generic.Options, args *Args) (*pipelineList, error) {
	pList := &pipelineList{
		options: opts,
		args:    args,
	}
	rest := opts.RESTClientGetter()
	c, err := client.NewClient(rest)
	if err != nil {
		return nil, err
	}
	pList.Client = c
	return pList, nil
}

// PipelineRunValue represents a single pipeline run.
type PipelineRunValue struct {
	Name              string
	CreationTimestamp metav1.Time
	Namespace         string
	CreatorPipeline   string
}

// ListExecute fetches and displays a formatted list of PipelineRuns.
func (p *pipelineList) ListExecute() error {
	listOpts := &ctrlclient.ListOptions{}

	// Apply namespace filter if AllNamespaces flag is not set.
	if !p.args.AllNamespaces {
		listOpts.Namespace = p.args.Namespace
	}

	pipelineRunList := &tektonapi.PipelineRunList{}
	if err := p.CtrlRuntimeClient().List(context.Background(), pipelineRunList, listOpts); err != nil {
		logrus.Errorf("failed to get PipelineRunList, %v", err)
		return err
	}

	// Transform pipelineRunList items to PipelineRunValue instances.
	var valueList []PipelineRunValue
	for _, tr := range pipelineRunList.Items {
		valueList = append(valueList, PipelineRunValue{
			Name:              tr.Name,
			CreationTimestamp: tr.CreationTimestamp,
			Namespace:         tr.Namespace,
			CreatorPipeline:   tr.Spec.PipelineRef.Name,
		})
	}

	// Group and sort pipeline runs for display.
	groupedRuns := GroupAndSortPipelineRuns(valueList)

	fmt.Println("------------------------------------- Pipeline Execution -----------------------------")
	fmt.Println("  Execution Name          |   Creation Time     |   Namespace      | Creator Pipeline")
	fmt.Println("--------------------------------------------------------------------------------------")

	for _, runs := range groupedRuns {
		for _, tr := range runs {
			fmt.Printf("%-25s | %-20s | %-16s | %s\n",
				tr.Name,
				tr.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
				tr.Namespace,
				tr.CreatorPipeline)
		}
	}

	return nil
}

// GroupAndSortPipelineRuns organizes PipelineRunValues by CreatorPipeline and orders them by CreationTimestamp within each group.
func GroupAndSortPipelineRuns(runs []PipelineRunValue) map[string][]PipelineRunValue {
	groups := make(map[string][]PipelineRunValue)
	for _, run := range runs {
		groups[run.CreatorPipeline] = append(groups[run.CreatorPipeline], run)
	}

	// Sort each group by creation timestamp.
	for _, group := range groups {
		sort.Slice(group, func(i, j int) bool {
			return group[i].CreationTimestamp.Time.Before(group[j].CreationTimestamp.Time)
		})
	}

	return groups
}
