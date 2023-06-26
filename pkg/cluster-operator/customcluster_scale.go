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

package clusteroperator

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"kurator.dev/kurator/pkg/apis/infra/v1alpha1"
)

// reconcileScaleUp is responsible for handling the customCluster reconciliation process when worker nodes need to be scaled up.
func (r *CustomClusterController) reconcileScaleUp(ctx context.Context, customCluster *v1alpha1.CustomCluster, scaleUpWorkerNodes []NodeInfo, kcp *controlplanev1.KubeadmControlPlane) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Create a temporary configmap that represents the desired state to create the scaleUp pod.
	if _, err := r.ensureScaleUpHostsCreated(ctx, customCluster, scaleUpWorkerNodes); err != nil {
		log.Error(err, "failed to ensure that scaleUp configmap is created ", "name", customCluster.Name, "namespace", customCluster.Namespace)
		return ctrl.Result{}, err
	}

	// Create the scaleUp pod.
	workerPod, err1 := r.ensureWorkerPodCreated(ctx, customCluster, CustomClusterScaleUpAction, KubesprayScaleUpCMD, generateScaleUpHostsName(customCluster), generateClusterConfigName(customCluster), kcp.Spec.Version)
	if err1 != nil {
		conditions.MarkFalse(customCluster, v1alpha1.ScaledUpCondition, v1alpha1.FailedCreateScaleUpWorker,
			clusterv1.ConditionSeverityWarning, "scale up worker is failed to create %s/%s.", customCluster.Namespace, customCluster.Name)
		log.Error(err1, "failed to ensure that scaleUp WorkerPod is created ", "name", customCluster.Name, "namespace", customCluster.Namespace)
		return ctrl.Result{}, err1
	}

	// Check the current customCluster status.
	if customCluster.Status.Phase != v1alpha1.ScalingUpPhase {
		log.Info("phase changes", "prevPhase", customCluster.Status.Phase, "currentPhase", v1alpha1.ScalingUpPhase)
		customCluster.Status.Phase = v1alpha1.ScalingUpPhase
	}

	// Determine the progress of scaling based on the status of the workerPod.
	if workerPod.Status.Phase == corev1.PodSucceeded {
		// Update cluster nodes to ensure that the current cluster-host represents the cluster after the scaling.
		if err := r.updateClusterNodes(ctx, customCluster, scaleUpWorkerNodes); err != nil {
			log.Error(err, "failed to update cluster nodes")
			return ctrl.Result{}, err
		}

		// The scale up process is completed by restoring the workerPod's status to "provisioned".
		log.Info("phase changes", "prevPhase", customCluster.Status.Phase, "currentPhase", v1alpha1.ProvisionedPhase)
		customCluster.Status.Phase = v1alpha1.ProvisionedPhase

		// Delete the temporary scaleUp cm.
		if err := r.ensureConfigMapDeleted(ctx, generateScaleUpHostsKey(customCluster)); err != nil {
			log.Error(err, "failed to delete scaleUp configmap", "configmap", generateScaleUpHostsKey(customCluster))
			return ctrl.Result{}, err
		}

		// Delete the scaleUp worker.
		if err := r.ensureWorkerPodDeleted(ctx, customCluster, CustomClusterScaleUpAction); err != nil {
			log.Error(err, "failed to delete scaleUp worker pod", "customCluster", customCluster.Name)
			return ctrl.Result{}, err
		}
		conditions.MarkTrue(customCluster, v1alpha1.ScaledUpCondition)
		return ctrl.Result{}, nil
	}

	// When scaleUp worker pod runs failed, the status of customCluster will change into "provisioned". Deleting this error one will trigger the creation of a new scaleUp worker pod.
	if workerPod.Status.Phase == corev1.PodFailed {
		log.Info("scale up failed, phase changes", "prevPhase", customCluster.Status.Phase, "currentPhase", v1alpha1.ProvisionedPhase)
		customCluster.Status.Phase = v1alpha1.ProvisionedPhase

		conditions.MarkFalse(customCluster, v1alpha1.ScaledUpCondition, v1alpha1.ScaleUpWorkerRunFailedReason,
			clusterv1.ConditionSeverityWarning, "scale up worker run failed %s/%s", customCluster.Namespace, customCluster.Name)
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// reconcileScaleDown is responsible for handling the customCluster reconciliation process when worker nodes need to be scaled down.
func (r *CustomClusterController) reconcileScaleDown(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, scaleDownWorkerNodes []NodeInfo, kcp *controlplanev1.KubeadmControlPlane) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Checks whether the worker node for scaling down already exists. If it does not exist, the function creates it.
	workerPod, err1 := r.ensureWorkerPodCreated(ctx, customCluster, CustomClusterScaleDownAction, generateScaleDownManageCMD(scaleDownWorkerNodes), generateClusterHostsName(customCluster), generateClusterConfigName(customCluster), kcp.Spec.Version)
	if err1 != nil {
		conditions.MarkFalse(customCluster, v1alpha1.ScaledDownCondition, v1alpha1.FailedCreateScaleDownWorker,
			clusterv1.ConditionSeverityWarning, "scale down worker is failed to create %s/%s.", customCluster.Namespace, customCluster.Name)
		log.Error(err1, "failed to ensure that scaleDown WorkerPod is created", "name", customCluster.Name, "namespace", customCluster.Namespace)
		return ctrl.Result{}, err1
	}

	// Check the current customCluster status.
	if customCluster.Status.Phase != v1alpha1.ScalingDownPhase {
		log.Info("phase changes", "prevPhase", customCluster.Status.Phase, "currentPhase", v1alpha1.ScalingDownPhase)
		customCluster.Status.Phase = v1alpha1.ScalingDownPhase
	}

	// Determine the progress of scaling based on the status of the workerPod.
	if workerPod.Status.Phase == corev1.PodSucceeded {
		// Recreate the cluster-hosts to ensure that the current cluster-host represents the cluster after the deletion.
		if _, err := r.recreateClusterHosts(ctx, customCluster, customMachine); err != nil {
			log.Error(err, "failed to recreate configmap cluster-hosts when scale down")
			return ctrl.Result{}, err
		}

		// The scale down process is completed by restoring the workerPod's status to "provisioned".
		log.Info("phase changes", "prevPhase", customCluster.Status.Phase, "currentPhase", v1alpha1.ProvisionedPhase)
		customCluster.Status.Phase = v1alpha1.ProvisionedPhase

		// Delete the scaleDown worker
		if err := r.ensureWorkerPodDeleted(ctx, customCluster, CustomClusterScaleDownAction); err != nil {
			log.Error(err, "failed to delete scaleDown worker pod", "worker", customCluster.Name)
			return ctrl.Result{}, err
		}
		conditions.MarkTrue(customCluster, v1alpha1.ScaledDownCondition)

		return ctrl.Result{}, nil
	}

	// When scaleDown worker pod runs failed, the status of customCluster will change into "provisioned". Deleting this error one will trigger the creation of a new scaleDown worker pod.
	if workerPod.Status.Phase == corev1.PodFailed {
		log.Info("scale down failed, phase changes", "prevPhase", customCluster.Status.Phase, "currentPhase", v1alpha1.ProvisionedPhase)
		customCluster.Status.Phase = v1alpha1.ProvisionedPhase

		conditions.MarkFalse(customCluster, v1alpha1.ScaledDownCondition, v1alpha1.ScaleDownWorkerRunFailedReason,
			clusterv1.ConditionSeverityWarning, "scale down worker run failed %s/%s", customCluster.Namespace, customCluster.Name)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// createScaleUpConfigMap create temporary cluster-hosts configmap for scaling.
func (r *CustomClusterController) createScaleUpConfigMap(ctx context.Context, customCluster *v1alpha1.CustomCluster, scaleUpWorkerNodes []NodeInfo) (*corev1.ConfigMap, error) {
	// Get current cm.
	curCM := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, generateClusterHostsKey(customCluster), curCM); err != nil {
		return nil, err
	}

	newCM := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      generateScaleUpHostsName(customCluster),
			Namespace: customCluster.Namespace,
		},
		Data: map[string]string{ClusterHostsName: strings.TrimSpace(getScaleUpConfigMapData(curCM.Data[ClusterHostsName], scaleUpWorkerNodes))},
	}

	if err := r.Client.Create(ctx, newCM); err != nil {
		return nil, err
	}
	return newCM, nil
}

// ensureScaleUpHostsCreated ensure that the temporary cluster-hosts configmap for scaling up is created.
func (r *CustomClusterController) ensureScaleUpHostsCreated(ctx context.Context, customCluster *v1alpha1.CustomCluster, scaleUpWorkerNodes []NodeInfo) (*corev1.ConfigMap, error) {
	cmKey := generateScaleUpHostsKey(customCluster)
	cm := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, cmKey, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return r.createScaleUpConfigMap(ctx, customCluster, scaleUpWorkerNodes)
		}
		return nil, err
	}
	return cm, nil
}

// findScaleUpWorkerNodes find the workerNodes which need to be scale up.
func findScaleUpWorkerNodes(provisionedWorkerNodes, curWorkerNodes []NodeInfo) []NodeInfo {
	return findAdditionalWorkerNodes(provisionedWorkerNodes, curWorkerNodes)
}

// findScaleDownWorkerNodes find the workerNodes which need to be scale down.
func findScaleDownWorkerNodes(provisionedWorkerNodes, curWorkerNodes []NodeInfo) []NodeInfo {
	return findAdditionalWorkerNodes(curWorkerNodes, provisionedWorkerNodes)
}

// findAdditionalWorkerNodes find additional workers in secondWorkersNodes than firstWorkerNodes.
func findAdditionalWorkerNodes(firstWorkerNodes, secondWorkersNodes []NodeInfo) []NodeInfo {
	var additionalWorkers []NodeInfo
	if len(secondWorkersNodes) == 0 {
		return nil
	}
	if len(firstWorkerNodes) == 0 {
		additionalWorkers = append(additionalWorkers, secondWorkersNodes...)
		return additionalWorkers
	}
	var set = make(map[string]struct{})

	for _, firstNode := range firstWorkerNodes {
		set[firstNode.NodeName] = struct{}{}
	}

	for _, secondNode := range secondWorkersNodes {
		if _, ok := set[secondNode.NodeName]; !ok {
			additionalWorkers = append(additionalWorkers, secondNode)
		}
	}

	return additionalWorkers
}

func generateScaleUpHostsKey(customCluster *v1alpha1.CustomCluster) client.ObjectKey {
	return client.ObjectKey{
		Namespace: customCluster.Namespace,
		Name:      generateScaleUpHostsName(customCluster),
	}
}

func generateScaleUpHostsName(customCluster *v1alpha1.CustomCluster) string {
	return customCluster.Name + "-" + ClusterHostsName + "-scale-up"
}

// generateScaleDownManageCMD generate a kubespray cmd to delete the node from the list of nodesNeedDelete.
func generateScaleDownManageCMD(nodesNeedDelete []NodeInfo) customClusterManageCMD {
	if len(nodesNeedDelete) == 0 {
		return ""
	}
	cmd := string(KubesprayScaleDownCMDPrefix) + " --extra-vars \"node=" + nodesNeedDelete[0].NodeName
	if len(nodesNeedDelete) == 1 {
		return customClusterManageCMD(cmd + "\" ")
	}

	for i := 1; i < len(nodesNeedDelete); i++ {
		cmd = cmd + "," + nodesNeedDelete[i].NodeName
	}

	return customClusterManageCMD(cmd + "\" ")
}

// updateClusterNodes update the cluster nodes info in configmap.
func (r *CustomClusterController) updateClusterNodes(ctx context.Context, customCluster *v1alpha1.CustomCluster, scaleUpWorkerNodes []NodeInfo) error {
	// Get cm.
	cm := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, generateClusterHostsKey(customCluster), cm); err != nil {
		return err
	}

	// Add new nodes on the original data, instead of directly modifying it to the desired state (read from customMachine).
	// This is the basis for the automatic execution of scaleDown after scaleUp is completed when both "scaleUpWorkerNodes" and "scaleDownWorkerNodes" are not nil.
	cm.Data[ClusterHostsName] = getScaleUpConfigMapData(cm.Data[ClusterHostsName], scaleUpWorkerNodes)

	// Update cm.
	if err := r.Client.Update(ctx, cm); err != nil {
		return err
	}
	return nil
}

// getScaleUpConfigMapData return a string of the configmap data that adds the scaleUp nodes to the original data.
func getScaleUpConfigMapData(data string, scaleUpWorkerNodes []NodeInfo) string {
	sep := regexp.MustCompile(`\[kube_control_plane]|\[k8s-cluster:children]`)
	dateParts := sep.Split(data, -1)

	nodeAndIP := "\n"
	nodeName := "\n"

	for _, node := range scaleUpWorkerNodes {
		nodeAndIP = nodeAndIP + fmt.Sprintf("%s ansible_host=%s ip=%s\n", node.NodeName, node.PublicIP, node.PrivateIP)
		nodeName = nodeName + fmt.Sprintf("%s\n", node.NodeName)
	}

	ans := fmt.Sprintf("%s%s[kube_control_plane]%s%s[k8s-cluster:children]%s", dateParts[0], nodeAndIP, dateParts[1], nodeName, dateParts[2])

	return ans
}

// getWorkerNodeInfoFromClusterHost get the provisioned workerNode info on VMs from the cluster-host configmap.
func getWorkerNodeInfoFromClusterHosts(clusterHost *corev1.ConfigMap) []NodeInfo {
	var workerNodes []NodeInfo
	var allNodes = make(map[string]NodeInfo)

	clusterHostDate := clusterHost.Data[ClusterHostsName]
	clusterHostDate = strings.TrimSpace(clusterHostDate)

	// The regexp string depend on the template text which the function "CreateClusterHosts" use.
	sep := regexp.MustCompile(`\[all]|\[kube_control_plane]|\[kube_node]|\[k8s-cluster:children]`)
	clusterHostDateArr := sep.Split(clusterHostDate, -1)

	allNodesStr := clusterHostDateArr[1]
	workerNodesStr := clusterHostDateArr[3]

	zp := regexp.MustCompile(`[\t\n\f\r]`)
	allNodeArr := zp.Split(allNodesStr, -1)
	workerNodesArr := zp.Split(workerNodesStr, -1)

	// Get all nodes' info.
	for _, nodeStr := range allNodeArr {
		if len(nodeStr) == 0 {
			continue
		}
		nodeStr = strings.TrimSpace(nodeStr)
		curName, cruNodeInfo := getNodeInfoFromNodeStr(nodeStr)
		// Deduplication.
		allNodes[curName] = cruNodeInfo
	}

	// Choose workerNode from all node.
	for _, workerNodeName := range workerNodesArr {
		if len(workerNodeName) == 0 {
			continue
		}
		workerNodeName = strings.TrimSpace(workerNodeName)
		workerNodes = append(workerNodes, allNodes[workerNodeName])
	}

	return workerNodes
}

func getNodeInfoFromNodeStr(nodeStr string) (hostName string, nodeInfo NodeInfo) {
	nodeStr = strings.TrimSpace(nodeStr)
	// The sepStr depend on the template text which the function "CreateClusterHosts" use.
	sepStr := regexp.MustCompile(` ansible_host=| ip=`)
	strArr := sepStr.Split(nodeStr, -1)

	hostName = strArr[0]
	publicIP := strArr[1]
	privateIP := strArr[2]

	return hostName, NodeInfo{
		NodeName:  hostName,
		PublicIP:  publicIP,
		PrivateIP: privateIP,
	}
}

func getWorkerNodesFromCustomMachine(customMachine *v1alpha1.CustomMachine) []NodeInfo {
	var workerNodes []NodeInfo
	for i := 0; i < len(customMachine.Spec.Nodes); i++ {
		curNode := NodeInfo{
			NodeName:  customMachine.Spec.Nodes[i].HostName,
			PublicIP:  customMachine.Spec.Nodes[i].PublicIP,
			PrivateIP: customMachine.Spec.Nodes[i].PrivateIP,
		}
		workerNodes = append(workerNodes, curNode)
	}
	return workerNodes
}
