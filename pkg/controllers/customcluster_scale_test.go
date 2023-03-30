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
	scaleUpNodes1 := findScaleUpWorkerNodes(provisionedNodes, curNodes1)
	assert.Equal(t, []NodeInfo{workerNode2}, scaleUpNodes1)

	scaleUpNodes2 := findScaleUpWorkerNodes(provisionedNodes, curNodes2)
	assert.Equal(t, []NodeInfo{workerNode2}, scaleUpNodes2)

	scaleUpNodes3 := findScaleUpWorkerNodes(provisionedNodes, curNodes3)
	assert.Equal(t, 0, len(scaleUpNodes3))

	scaleUpNodes4 := findScaleUpWorkerNodes(nil, curNodes2)
	assert.Equal(t, curNodes2, scaleUpNodes4)

	scaleUpNodes5 := findScaleUpWorkerNodes(curNodes2, nil)
	assert.Equal(t, 0, len(scaleUpNodes5))
}

func TestFindScaleDownWorkerNodes(t *testing.T) {
	scaleDoneNodes1 := findScaleDownWorkerNodes(provisionedNodes, curNodes1)
	assert.Equal(t, []NodeInfo{workerNode1}, scaleDoneNodes1)

	scaleDoneNodes2 := findScaleDownWorkerNodes(provisionedNodes, curNodes2)
	assert.Equal(t, 0, len(scaleDoneNodes2))

	scaleDoneNodes3 := findScaleDownWorkerNodes(provisionedNodes, curNodes3)
	assert.Equal(t, []NodeInfo{workerNode3}, scaleDoneNodes3)

	scaleDoneNodes4 := findScaleDownWorkerNodes(nil, curNodes2)
	assert.Equal(t, 0, len(scaleDoneNodes4))

	scaleDoneNodes5 := findScaleDownWorkerNodes(curNodes2, nil)
	assert.Equal(t, curNodes2, scaleDoneNodes5)
}

func TestGenerateScaleDownManageCMD(t *testing.T) {
	scaleDownCMD1 := generateScaleDownManageCMD(nodeNeedDelete1)
	assert.Equal(t, customClusterManageCMD(""), scaleDownCMD1)

	scaleDownCMD2 := generateScaleDownManageCMD(nodeNeedDelete2)
	assert.Equal(t, customClusterManageCMD("ansible-playbook -i inventory/cluster-hosts --private-key /root/.ssh/ssh-privatekey remove-node.yml -vvv -e skip_confirmation=yes --extra-vars \"node=node1\" "), scaleDownCMD2)

	scaleDownCMD3 := generateScaleDownManageCMD(nodeNeedDelete3)
	assert.Equal(t, customClusterManageCMD("ansible-playbook -i inventory/cluster-hosts --private-key /root/.ssh/ssh-privatekey remove-node.yml -vvv -e skip_confirmation=yes --extra-vars \"node=node1,node2,node3\" "), scaleDownCMD3)
}

func TestGetScaleUpConfigMapData(t *testing.T) {
	ans := getScaleUpConfigMapData(clusterHostDataStr1, curNodes1)
	assert.Equal(t, clusterHostDataStr3, ans)
}
