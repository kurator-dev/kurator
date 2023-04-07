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
	corev1 "k8s.io/api/core/v1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"

	"kurator.dev/kurator/pkg/apis/infra/v1alpha1"
)

var master1 = v1alpha1.Machine{
	HostName:  "master1",
	PrivateIP: "1.1.1.1",
	PublicIP:  "2.2.2.2",
}

var node1 = v1alpha1.Machine{
	HostName:  "node1",
	PrivateIP: "3.3.3.3",
	PublicIP:  "4.4.4.4",
}

var node2 = v1alpha1.Machine{
	HostName:  "node2",
	PrivateIP: "5.5.5.5",
	PublicIP:  "6.6.6.6",
}

var curCustomMachineSingle = &v1alpha1.CustomMachine{
	Spec: v1alpha1.CustomMachineSpec{
		Master: []v1alpha1.Machine{master1},
		Nodes:  []v1alpha1.Machine{node1},
	},
}

var curCustomMachineMulti = &v1alpha1.CustomMachine{
	Spec: v1alpha1.CustomMachineSpec{
		Master: []v1alpha1.Machine{master1},
		Nodes:  []v1alpha1.Machine{node1, node2},
	},
}

func TestGetHostsContent(t *testing.T) {
	expectHost1 := &HostTemplateContent{
		NodeAndIP:    []string{"master1 ansible_host=2.2.2.2 ip=1.1.1.1", "node1 ansible_host=4.4.4.4 ip=3.3.3.3"},
		MasterName:   []string{"master1"},
		NodeName:     []string{"node1"},
		EtcdNodeName: []string{"master1"},
	}
	assert.Equal(t, expectHost1, GetHostsContent(curCustomMachineSingle))

	expectHost2 := &HostTemplateContent{
		NodeAndIP:    []string{"master1 ansible_host=2.2.2.2 ip=1.1.1.1", "node1 ansible_host=4.4.4.4 ip=3.3.3.3", "node2 ansible_host=6.6.6.6 ip=5.5.5.5"},
		MasterName:   []string{"master1"},
		NodeName:     []string{"node1", "node2"},
		EtcdNodeName: []string{"master1"},
	}

	assert.Equal(t, expectHost2, GetHostsContent(curCustomMachineMulti))
}

var targetWorkerNodesSingle = []NodeInfo{
	{
		NodeName:  "node1",
		PrivateIP: "3.3.3.3",
		PublicIP:  "4.4.4.4",
	},
}

var targetClusterInfoSingle = &ClusterInfo{
	WorkerNodes: targetWorkerNodesSingle,
}

var targetWorkerNodesMulti = []NodeInfo{
	{
		NodeName:  "node1",
		PrivateIP: "3.3.3.3",
		PublicIP:  "4.4.4.4",
	},
	{
		NodeName:  "node2",
		PrivateIP: "5.5.5.5",
		PublicIP:  "6.6.6.6",
	},
}

var targetClusterInfoMulti = &ClusterInfo{
	WorkerNodes: targetWorkerNodesMulti,
}

func TestGetWorkerNodesFromCustomMachine(t *testing.T) {
	workerNodes1 := getWorkerNodesFromCustomMachine(curCustomMachineSingle)
	assert.Equal(t, targetWorkerNodesSingle, workerNodes1)

	workerNodes2 := getWorkerNodesFromCustomMachine(curCustomMachineMulti)
	assert.Equal(t, targetWorkerNodesMulti, workerNodes2)
}

func TestDesiredClusterInfo(t *testing.T) {
	kcp1 := &controlplanev1.KubeadmControlPlane{
		Spec: controlplanev1.KubeadmControlPlaneSpec{
			Version: "v1.20.0",
		},
	}
	kcp2 := &controlplanev1.KubeadmControlPlane{
		Spec: controlplanev1.KubeadmControlPlaneSpec{
			Version: "v1.25.0",
		},
	}

	clusterInfo1 := getDesiredClusterInfo(curCustomMachineSingle, kcp1)
	assert.Equal(t, targetClusterInfoSingle, clusterInfo1)

	clusterInfo2 := getDesiredClusterInfo(curCustomMachineMulti, kcp2)
	assert.Equal(t, targetClusterInfoMulti, clusterInfo2)
}

var workerNode1 = NodeInfo{
	NodeName:  "node1",
	PublicIP:  "200.1.1.1",
	PrivateIP: "127.1.1.1",
}

var workerNode2 = NodeInfo{
	NodeName:  "node2",
	PublicIP:  "200.1.1.2",
	PrivateIP: "127.1.1.2",
}

var workerNode3 = NodeInfo{
	NodeName:  "node3",
	PublicIP:  "200.1.1.3",
	PrivateIP: "127.1.1.3",
}

var provisionedNodes = []NodeInfo{workerNode1, workerNode3}

var curNodes1 = []NodeInfo{workerNode2, workerNode3}

var curNodes2 = []NodeInfo{workerNode2, workerNode3, workerNode1}

var curNodes3 = []NodeInfo{workerNode1}

var nodeNeedDelete1 []NodeInfo
var nodeNeedDelete2 = []NodeInfo{workerNode1}
var nodeNeedDelete3 = []NodeInfo{workerNode1, workerNode2, workerNode3}

var clusterHostDataStr1 = "[all]\n\nmaster1 ansible_host=200.1.1.0 ip=127.1.1.0\n\nnode1 ansible_host=200.1.1.1 ip=127.1.1.1\n\n[kube_control_plane]\n\nmaster1\n\n[etcd]\nmaster1\n[kube_node]\nnode1\n[k8s-cluster:children]\nkube_node\nkube_control_plane"
var clusterHostDataStr2 = "[all]\n\nmaster1 ansible_host=200.1.1.0 ip=127.1.1.0\n\nnode1 ansible_host=200.1.1.1 ip=127.1.1.1\n\n[kube_control_plane]\n\nmaster1\n\n[etcd]\nmaster1\n[kube_node]\n\n[k8s-cluster:children]\nkube_node\nkube_control_plane"
var clusterHostDataStr3 = "[all]\n\nmaster1 ansible_host=200.1.1.0 ip=127.1.1.0\n\nnode1 ansible_host=200.1.1.1 ip=127.1.1.1\n\n\nnode2 ansible_host=200.1.1.2 ip=127.1.1.2\nnode3 ansible_host=200.1.1.3 ip=127.1.1.3\n[kube_control_plane]\n\nmaster1\n\n[etcd]\nmaster1\n[kube_node]\nnode1\n\nnode2\nnode3\n[k8s-cluster:children]\nkube_node\nkube_control_plane"

var clusterHost1 = &corev1.ConfigMap{
	Data: map[string]string{
		ClusterHostsName: clusterHostDataStr1,
	},
}

var clusterHost2 = &corev1.ConfigMap{
	Data: map[string]string{
		ClusterHostsName: clusterHostDataStr2,
	},
}

var clusterHost3 = &corev1.ConfigMap{
	Data: map[string]string{
		ClusterHostsName: clusterHostDataStr3,
	},
}

var masterNode = NodeInfo{
	NodeName:  "master1",
	PublicIP:  "200.1.1.0",
	PrivateIP: "127.1.1.0",
}

func TestGetWorkerNodeInfoFromClusterHost(t *testing.T) {
	nodeInfoArr1 := getWorkerNodeInfoFromClusterHosts(clusterHost1)
	assert.Equal(t, []NodeInfo{workerNode1}, nodeInfoArr1)

	nodeInfoArr2 := getWorkerNodeInfoFromClusterHosts(clusterHost2)
	assert.Equal(t, 0, len(nodeInfoArr2))

	nodeInfoArr3 := getWorkerNodeInfoFromClusterHosts(clusterHost3)
	assert.Equal(t, []NodeInfo{workerNode1, workerNode2, workerNode3}, nodeInfoArr3)
}

var nodeStr1 = "master1 ansible_host=200.1.1.0 ip=127.1.1.0"
var nodeStr2 = "node1 ansible_host=200.1.1.1 ip=127.1.1.1"

func TestGetNodeInfoFromNodeStr(t *testing.T) {
	hostName1, nodeInfo1 := getNodeInfoFromNodeStr(nodeStr1)
	assert.Equal(t, "master1", hostName1)
	assert.Equal(t, masterNode, nodeInfo1)

	hostName2, nodeInfo2 := getNodeInfoFromNodeStr(nodeStr2)
	assert.Equal(t, "node1", hostName2)
	assert.Equal(t, workerNode1, nodeInfo2)
}
