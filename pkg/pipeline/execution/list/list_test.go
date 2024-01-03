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

package list

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestGroupAndSortPipelineRuns tests the GroupAndSortPipelineRuns function.
func TestGroupAndSortPipelineRuns(t *testing.T) {
	tests := []struct {
		name     string
		runs     []PipelineRunValue
		expected map[string][]PipelineRunValue
	}{
		{
			name: "Group and sort pipeline runs",
			runs: []PipelineRunValue{
				{
					Name:              "run1",
					CreationTimestamp: metav1.Time{Time: time.Date(2024, 01, 02, 10, 00, 00, 00, time.UTC)},
					Namespace:         "ns1",
					CreatorPipeline:   "pipeline1",
				},
				{
					Name:              "run2",
					CreationTimestamp: metav1.Time{Time: time.Date(2024, 01, 02, 12, 00, 00, 00, time.UTC)},
					Namespace:         "ns2",
					CreatorPipeline:   "pipeline2",
				},
				{
					Name:              "run3",
					CreationTimestamp: metav1.Time{Time: time.Date(2024, 01, 02, 11, 00, 00, 00, time.UTC)},
					Namespace:         "ns1",
					CreatorPipeline:   "pipeline1",
				},
			},
			expected: map[string][]PipelineRunValue{
				"pipeline1": {
					{
						Name:              "run1",
						CreationTimestamp: metav1.Time{Time: time.Date(2024, 01, 02, 10, 00, 00, 00, time.UTC)},
						Namespace:         "ns1",
						CreatorPipeline:   "pipeline1",
					},
					{
						Name:              "run3",
						CreationTimestamp: metav1.Time{Time: time.Date(2024, 01, 02, 11, 00, 00, 00, time.UTC)},
						Namespace:         "ns1",
						CreatorPipeline:   "pipeline1",
					},
				},
				"pipeline2": {
					{
						Name:              "run2",
						CreationTimestamp: metav1.Time{Time: time.Date(2024, 01, 02, 12, 00, 00, 00, time.UTC)},
						Namespace:         "ns2",
						CreatorPipeline:   "pipeline2",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GroupAndSortPipelineRuns(tt.runs)
			assert.Equal(t, tt.expected, result)
		})
	}
}
