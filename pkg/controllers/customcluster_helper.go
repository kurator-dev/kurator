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
	"context"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"kurator.dev/kurator/pkg/apis/infra/v1alpha1"
)

// generateClusterManageWorker generate a kubespray manage worker pod from configmap.
func generateClusterManageWorker(customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction, manageCMD customClusterManageCMD, hostsName, configName string) *corev1.Pod {
	podName := customCluster.Name + "-" + string(manageAction)
	namespace := customCluster.Namespace
	defaultMode := int32(0o600)
	kubesprayImage := getKubesprayImage(DefaultKubesprayVersion)
	managerWorker := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      podName,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},

		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    podName,
					Image:   kubesprayImage,
					Command: []string{"/bin/sh", "-c"},
					Args:    []string{string(manageCMD)},

					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      ClusterHostsName,
							MountPath: "/kubespray/inventory",
						},
						{
							Name:      ClusterConfigName,
							MountPath: "/kubespray/inventory/group_vars/all",
						},
						{
							Name:      SecreteName,
							MountPath: "/root/.ssh",
							ReadOnly:  true,
						},
					},
				},
			},

			Volumes: []corev1.Volume{
				{
					Name: ClusterHostsName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: hostsName,
							},
						},
					},
				},
				{
					Name: ClusterConfigName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configName,
							},
						},
					},
				},
				{
					Name: SecreteName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  SecreteName,
							DefaultMode: &defaultMode,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
	return managerWorker
}

type HostTemplateContent struct {
	NodeAndIP    []string
	MasterName   []string
	NodeName     []string
	EtcdNodeName []string // default: NodeName + MasterName
}

type ConfigTemplateContent struct {
	KubeVersion string
	// The default value is 10.233.0.0/18, must be unused block of space.
	ServiceCIDR string
	// The default value is 10.233.64.0/18, must be unused in your network infrastructure.
	PodCIDR string
	// CNIType is the CNI plugin of the cluster on VMs. The default plugin is calico and can be ["calico", "cilium", "canal", "flannel"]
	CNIType string
	// ControlPlaneConfigAddress same as `ControlPlaneEndpoint`.
	ControlPlaneAddress string
	// ControlPlaneCertSANs sets extra Subject Alternative Names for the API Server signing cert.
	ControlPlaneCertSANs string
	ClusterName          string
	DnsDomain            string
	KubeImageRepo        string
	// FeatureGates is a map that stores the names and boolean values of Kubernetes feature gates.
	// The keys of the map are the names of the feature gates, and the values are boolean values that indicate whether
	// the feature gate is enabled (true) or disabled (false).
	FeatureGates map[string]bool
	// LBDomainName is a variable used to set the endpoint for a Kubernetes cluster when a load balancer is enabled.
	LBDomainName string
	// TODO: support other kubernetes configs
}

func GetConfigContent(c *clusterv1.Cluster, kcp *controlplanev1.KubeadmControlPlane, cc *v1alpha1.CustomCluster) *ConfigTemplateContent {
	// Add kubespray init config here
	configContent := &ConfigTemplateContent{
		PodCIDR:              c.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		ServiceCIDR:          c.Spec.ClusterNetwork.Services.CIDRBlocks[0],
		KubeVersion:          kcp.Spec.Version,
		CNIType:              cc.Spec.CNI.Type,
		ControlPlaneAddress:  cc.Spec.ControlPlaneConfig.Address,
		ControlPlaneCertSANs: strings.Join(cc.Spec.ControlPlaneConfig.CertSANs, ","),
		ClusterName:          kcp.Spec.KubeadmConfigSpec.ClusterConfiguration.ClusterName,
		DnsDomain:            c.Spec.ClusterNetwork.ServiceDomain,
		KubeImageRepo:        kcp.Spec.KubeadmConfigSpec.ClusterConfiguration.ImageRepository,
		FeatureGates:         kcp.Spec.KubeadmConfigSpec.ClusterConfiguration.FeatureGates,
		LBDomainName:         cc.Spec.ControlPlaneConfig.LBDomainName,
	}
	return configContent
}

func GetHostsContent(customMachine *v1alpha1.CustomMachine) *HostTemplateContent {
	masterMachine := customMachine.Spec.Master
	nodeMachine := customMachine.Spec.Nodes
	hostVar := &HostTemplateContent{
		NodeAndIP:    make([]string, len(masterMachine)+len(nodeMachine)),
		MasterName:   make([]string, len(masterMachine)),
		NodeName:     make([]string, len(nodeMachine)),
		EtcdNodeName: make([]string, len(masterMachine)),
	}

	count := 0
	for i, machine := range masterMachine {
		masterName := machine.HostName
		nodeAndIp := fmt.Sprintf("%s ansible_host=%s ip=%s", machine.HostName, machine.PublicIP, machine.PrivateIP)
		hostVar.MasterName[i] = masterName
		hostVar.EtcdNodeName[count] = masterName
		hostVar.NodeAndIP[count] = nodeAndIp
		count++
	}
	for i, machine := range nodeMachine {
		nodeName := machine.HostName
		nodeAndIp := fmt.Sprintf("%s ansible_host=%s ip=%s", machine.HostName, machine.PublicIP, machine.PrivateIP)
		hostVar.NodeName[i] = nodeName
		hostVar.NodeAndIP[count] = nodeAndIp
		count++
	}

	return hostVar
}

func (r *CustomClusterController) CreateConfigMapWithTemplate(ctx context.Context, name, namespace, fileName, configMapData string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{fileName: strings.TrimSpace(configMapData)},
	}

	if err := r.Client.Create(ctx, cm); err != nil {
		return nil, err
	}
	return cm, nil
}

// recreateClusterHosts delete current clusterHosts configmap and create a new one with latest customMachine configuration.
func (r *CustomClusterController) recreateClusterHosts(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine) (*corev1.ConfigMap, error) {
	// Delete the configmap cluster-hosts.
	if err := r.ensureConfigMapDeleted(ctx, generateClusterHostsKey(customCluster)); err != nil {
		return nil, err
	}
	return r.CreateClusterHosts(ctx, customMachine, customCluster)
}

//go:embed customcluster_clusterhosts.template
var clusterHostsTemplate string

func (r *CustomClusterController) CreateClusterHosts(ctx context.Context, customMachine *v1alpha1.CustomMachine, customCluster *v1alpha1.CustomCluster) (*corev1.ConfigMap, error) {
	hostsContent := GetHostsContent(customMachine)
	hostData := &strings.Builder{}

	tmpl := template.Must(template.New("").Parse(clusterHostsTemplate))
	if err := tmpl.Execute(hostData, hostsContent); err != nil {
		return nil, err
	}
	hostsTemplate := hostData.String()

	name := generateClusterHostsName(customCluster)
	namespace := customCluster.Namespace

	return r.CreateConfigMapWithTemplate(ctx, name, namespace, ClusterHostsName, hostsTemplate)
}

//go:embed customcluster_clusterconfig.template
var clusterConfigTemplate string

func (r *CustomClusterController) CreateClusterConfig(ctx context.Context, c *clusterv1.Cluster, kcp *controlplanev1.KubeadmControlPlane, cc *v1alpha1.CustomCluster) (*corev1.ConfigMap, error) {
	configContent := GetConfigContent(c, kcp, cc)
	configData := &strings.Builder{}

	tmpl := template.Must(template.New("").Parse(clusterConfigTemplate))
	if err := tmpl.Execute(configData, configContent); err != nil {
		return nil, err
	}
	configTemplate := configData.String()
	name := generateClusterConfigName(cc)
	namespace := cc.Namespace

	return r.CreateConfigMapWithTemplate(ctx, name, namespace, ClusterConfigName, configTemplate)
}

func generateWorkerKey(customCluster *v1alpha1.CustomCluster, action customClusterManageAction) client.ObjectKey {
	return client.ObjectKey{
		Namespace: customCluster.Namespace,
		Name:      customCluster.Name + "-" + string(action),
	}
}

func generateClusterHostsKey(customCluster *v1alpha1.CustomCluster) client.ObjectKey {
	return client.ObjectKey{
		Namespace: customCluster.Namespace,
		Name:      generateClusterHostsName(customCluster),
	}
}

func generateClusterConfigKey(customCluster *v1alpha1.CustomCluster) client.ObjectKey {
	return client.ObjectKey{
		Namespace: customCluster.Namespace,
		Name:      generateClusterConfigName(customCluster),
	}
}

func generateClusterHostsName(customCluster *v1alpha1.CustomCluster) string {
	return customCluster.Name + "-" + ClusterHostsName
}

func generateClusterConfigName(customCluster *v1alpha1.CustomCluster) string {
	return customCluster.Name + "-" + ClusterConfigName
}

func generateOwnerRefFromCustomCluster(customCluster *v1alpha1.CustomCluster) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: customCluster.APIVersion,
		Kind:       customCluster.Kind,
		Name:       customCluster.Name,
		UID:        customCluster.UID,
	}
}

// getDesiredClusterInfo get desired cluster info from crd configuration.
func getDesiredClusterInfo(customMachine *v1alpha1.CustomMachine, kcp *controlplanev1.KubeadmControlPlane) *ClusterInfo {
	workerNodes := getWorkerNodesFromCustomMachine(customMachine)

	clusterInfo := &ClusterInfo{
		WorkerNodes: workerNodes,
		KubeVersion: kcp.Spec.Version,
	}

	return clusterInfo
}

// getProvisionedClusterInfo get the provisioned cluster info on VMs from current configmap.
func (r *CustomClusterController) getProvisionedClusterInfo(ctx context.Context, customCluster *v1alpha1.CustomCluster) (*ClusterInfo, error) {
	// get current cluster-host configMap
	clusterHosts := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, generateClusterHostsKey(customCluster), clusterHosts); err != nil {
		return nil, err
	}
	// get workerNode from cluster-host
	workerNodes := getWorkerNodeInfoFromClusterHosts(clusterHosts)

	// get current cluster-config configMap
	clusterConfig := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, generateClusterConfigKey(customCluster), clusterConfig); err != nil {
		return nil, err
	}

	// get provisioned version from cluster-config
	provisionedVersion := getKubeVersionFromCM(clusterConfig)

	// get the provisioned cluster info
	clusterInfo := &ClusterInfo{
		WorkerNodes: workerNodes,
		KubeVersion: provisionedVersion,
	}

	return clusterInfo, nil
}

// ensureClusterHostsCreated ensure that the cluster-hosts configmap is created.
func (r *CustomClusterController) ensureClusterHostsCreated(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine) (*corev1.ConfigMap, error) {
	cmKey := generateClusterHostsKey(customCluster)
	cm := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, cmKey, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return r.CreateClusterHosts(ctx, customMachine, customCluster)
		}
		return nil, err
	}
	return cm, nil
}

// ensureClusterConfigCreated ensure that the cluster-config configmap is created.
func (r *CustomClusterController) ensureClusterConfigCreated(ctx context.Context, customCluster *v1alpha1.CustomCluster, cc *v1alpha1.CustomCluster, cluster *clusterv1.Cluster, kcp *controlplanev1.KubeadmControlPlane) (*corev1.ConfigMap, error) {
	cmKey := generateClusterConfigKey(customCluster)
	cm := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, cmKey, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return r.CreateClusterConfig(ctx, cluster, kcp, cc)
		}
		return nil, err
	}
	return cm, nil
}

// ensureConfigMapDeleted ensure that the configmap is deleted.
func (r *CustomClusterController) ensureConfigMapDeleted(ctx context.Context, cmKey client.ObjectKey) error {
	cm := &corev1.ConfigMap{}
	errGetConfigmap := r.Client.Get(ctx, cmKey, cm)
	// errGetConfigmap can be divided into three situation: isNotFound(no action); not isNotFound(return err); nil(start to delete cm).
	if apierrors.IsNotFound(errGetConfigmap) {
		return nil
	} else if errGetConfigmap != nil && !apierrors.IsNotFound(errGetConfigmap) {
		return fmt.Errorf("failed to get cm when it should be deleted: %v", errGetConfigmap)
	} else if errGetConfigmap == nil {
		controllerutil.RemoveFinalizer(cm, CustomClusterConfigMapFinalizer)
		if err := r.Client.Update(ctx, cm); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to remove finalizer of cm when it should be deleted: %v", err)
		}
		if err := r.Client.Delete(ctx, cm); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete cm when it should be deleted: %v", err)
		}
	}
	return nil
}

// ensureWorkerPodDeleted ensures that the worker pod is deleted.
func (r *CustomClusterController) ensureWorkerPodDeleted(ctx context.Context, workerPodKey client.ObjectKey) error {
	worker := &corev1.Pod{}
	errGetWorker := r.Client.Get(ctx, workerPodKey, worker)
	// errGetWorker can be divided into three situation: isNotFound; not isNotFound; nil.
	if apierrors.IsNotFound(errGetWorker) {
		return nil
	} else if errGetWorker != nil && !apierrors.IsNotFound(errGetWorker) {
		return fmt.Errorf("failed to get worker pod when it should be deleted: %v", errGetWorker)
	} else if errGetWorker == nil {
		if err := r.Client.Delete(ctx, worker); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete cm when it should be deleted: %v", err)
		}
	}
	return nil
}

// ensureWorkerPodCreated ensure that the worker pod is created.
func (r *CustomClusterController) ensureWorkerPodCreated(ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction, manageCMD customClusterManageCMD, hostName, configName string) (*corev1.Pod, error) {
	workerPodKey := generateWorkerKey(customCluster, manageAction)
	workerPod := &corev1.Pod{}

	if err := r.Client.Get(ctx, workerPodKey, workerPod); err != nil {
		if apierrors.IsNotFound(err) {
			workerPod = generateClusterManageWorker(customCluster, manageAction, manageCMD, hostName, configName)
			workerPod.OwnerReferences = []metav1.OwnerReference{generateOwnerRefFromCustomCluster(customCluster)}
			if err1 := r.Client.Create(ctx, workerPod); err1 != nil {
				return nil, fmt.Errorf("failed to create customCluster manager worker pod: %v", err1)
			}
			return workerPod, nil
		}
		return nil, fmt.Errorf("failed to get worker pod: %v", err)
	}
	return workerPod, nil
}

// getKubesprayImage take in kubesprayVersion return the kubespray image url of this version.
func getKubesprayImage(kubesprayVersion string) string {
	imagePath := "quay.io/kubespray/kubespray:" + kubesprayVersion
	return imagePath
}

// hasProvisionClusterInfo is used to determine if the current phase is valid for retrieving ProvisionClusterInfo.
func hasProvisionClusterInfo(phase v1alpha1.CustomClusterPhase) bool {
	if phase == v1alpha1.ProvisionedPhase || phase == v1alpha1.ScalingDownPhase || phase == v1alpha1.ScalingUpPhase || phase == v1alpha1.UpgradingPhase {
		return true
	}
	return false
}
