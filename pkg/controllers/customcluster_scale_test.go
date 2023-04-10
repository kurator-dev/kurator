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

package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindScaleUpWorkerNodes(t *testing.T) {
	test := []struct {
		name             string
		provisionedNodes []NodeInfo
		desiredNodes     []NodeInfo
		expected         []NodeInfo
	}{
		{
			name:             "one node same",
			provisionedNodes: []NodeInfo{workerNode1, workerNode3},
			desiredNodes:     []NodeInfo{workerNode2, workerNode3},
			expected:         []NodeInfo{workerNode2},
		},
		{
			name:             "one node more",
			provisionedNodes: []NodeInfo{workerNode3},
			desiredNodes:     []NodeInfo{workerNode2, workerNode3},
			expected:         []NodeInfo{workerNode2},
		},
		{
			name:             "one node less",
			provisionedNodes: []NodeInfo{workerNode1, workerNode3},
			desiredNodes:     []NodeInfo{workerNode3},
			expected:         nil,
		},
	}

	for _, tc := range test {
		t.Run(tc.name, func(t *testing.T) {
			got := findScaleUpWorkerNodes(tc.provisionedNodes, tc.desiredNodes)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestFindScaleDownWorkerNodes(t *testing.T) {
	test := []struct {
		name             string
		provisionedNodes []NodeInfo
		desiredNodes     []NodeInfo
		expected         []NodeInfo
	}{
		{
			name:             "one node same",
			provisionedNodes: []NodeInfo{workerNode1, workerNode3},
			desiredNodes:     []NodeInfo{workerNode2, workerNode3},
			expected:         []NodeInfo{workerNode1},
		},
		{
			name:             "one node more",
			provisionedNodes: []NodeInfo{workerNode3},
			desiredNodes:     []NodeInfo{workerNode2, workerNode3},
			expected:         nil,
		},
		{
			name:             "one node less",
			provisionedNodes: []NodeInfo{workerNode1, workerNode3},
			desiredNodes:     []NodeInfo{workerNode3},
			expected:         []NodeInfo{workerNode1},
		},
	}

	for _, tc := range test {
		t.Run(tc.name, func(t *testing.T) {
			got := findScaleDownWorkerNodes(tc.provisionedNodes, tc.desiredNodes)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestGenerateScaleDownManageCMD(t *testing.T) {
	test := []struct {
		name           string
		nodeNeedDelete []NodeInfo
		expected       customClusterManageCMD
	}{
		{
			name:           "single node",
			nodeNeedDelete: []NodeInfo{workerNode1},
			expected:       customClusterManageCMD("ansible-playbook -i inventory/cluster-hosts --private-key /root/.ssh/ssh-privatekey remove-node.yml -vvv -e skip_confirmation=yes --extra-vars \"node=node1\" "),
		},
		{
			name:           "muilti node",
			nodeNeedDelete: []NodeInfo{workerNode1, workerNode2, workerNode3},
			expected:       customClusterManageCMD("ansible-playbook -i inventory/cluster-hosts --private-key /root/.ssh/ssh-privatekey remove-node.yml -vvv -e skip_confirmation=yes --extra-vars \"node=node1,node2,node3\" "),
		},
	}

	for _, tc := range test {
		t.Run(tc.name, func(t *testing.T) {
			got := generateScaleDownManageCMD(tc.nodeNeedDelete)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestGetScaleUpConfigMapData(t *testing.T) {
	test := []struct {
		name               string
		clusterHostDataStr string
		curNodes           []NodeInfo
		expected           string
	}{
		{
			name:               "add node: workerNode2, workerNode3",
			clusterHostDataStr: clusterHostDataStr1,
			curNodes:           []NodeInfo{workerNode2, workerNode3},
			expected:           clusterHostDataStr3,
		},
	}

	for _, tc := range test {
		t.Run(tc.name, func(t *testing.T) {
			got := getScaleUpConfigMapData(tc.clusterHostDataStr, tc.curNodes)
			assert.Equal(t, tc.expected, got)
		})
	}
}
