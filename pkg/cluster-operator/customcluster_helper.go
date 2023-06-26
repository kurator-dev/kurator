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
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/coreos/go-semver/semver"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/storage/names"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"kurator.dev/kurator/pkg/apis/infra/v1alpha1"
)

// generateClusterManageWorker generate a kubespray manage worker pod from configmap.
func generateClusterManageWorker(customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction, manageCMD customClusterManageCMD, hostsName, configName, kubesprayImage string) *corev1.Pod {
	basePodName := customCluster.Name + "-" + string(manageAction)
	podName := names.SimpleNameGenerator.GenerateName(basePodName + "-")
	namespace := customCluster.Namespace
	defaultMode := int32(0o600)
	managerWorker := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      podName,
			Labels:    map[string]string{ManageActionLabel: string(manageAction)},
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
	// LoadBalancerDomainName is a variable used to set the endpoint for a Kubernetes cluster when a load balancer is enabled.
	LoadBalancerDomainName string
	// TODO: support other kubernetes configs
}

func GetConfigContent(c *clusterv1.Cluster, kcp *controlplanev1.KubeadmControlPlane, cc *v1alpha1.CustomCluster) *ConfigTemplateContent {
	// Add kubespray init config here
	configContent := &ConfigTemplateContent{
		PodCIDR:                c.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		ServiceCIDR:            c.Spec.ClusterNetwork.Services.CIDRBlocks[0],
		KubeVersion:            kcp.Spec.Version,
		CNIType:                cc.Spec.CNI.Type,
		ControlPlaneAddress:    cc.Spec.ControlPlaneConfig.Address,
		ControlPlaneCertSANs:   strings.Join(cc.Spec.ControlPlaneConfig.CertSANs, ","),
		ClusterName:            kcp.Spec.KubeadmConfigSpec.ClusterConfiguration.ClusterName,
		DnsDomain:              c.Spec.ClusterNetwork.ServiceDomain,
		KubeImageRepo:          kcp.Spec.KubeadmConfigSpec.ClusterConfiguration.ImageRepository,
		FeatureGates:           kcp.Spec.KubeadmConfigSpec.ClusterConfiguration.FeatureGates,
		LoadBalancerDomainName: cc.Spec.ControlPlaneConfig.LoadBalancerDomainName,
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
func (r *CustomClusterController) ensureWorkerPodDeleted(ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction) error {
	workerPod, err := r.findManageWorkerPod(ctx, customCluster, manageAction)

	if err != nil {
		return fmt.Errorf("failed find customCluster manager worker pod: %v", err)
	}
	if workerPod == nil {
		return nil
	}

	if err := r.Client.Delete(ctx, workerPod); err != nil {
		return fmt.Errorf("failed to delete workerPod: %v", err)
	}

	return nil
}

// ensureWorkerPodCreated ensure that the worker pod is created.
func (r *CustomClusterController) ensureWorkerPodCreated(ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction, manageCMD customClusterManageCMD, hostName, configName, kubeVersion string) (*corev1.Pod, error) {
	workerPod, err := r.findManageWorkerPod(ctx, customCluster, manageAction)

	if err != nil {
		return nil, fmt.Errorf("failed find customCluster manager worker pod: %v", err)
	}
	if workerPod != nil {
		return workerPod, nil
	}

	kubesprayImage := getKubesprayImage(ctx, kubeVersion)

	newWorkerPod := generateClusterManageWorker(customCluster, manageAction, manageCMD, hostName, configName, kubesprayImage)
	newWorkerPod.OwnerReferences = []metav1.OwnerReference{generateOwnerRefFromCustomCluster(customCluster)}
	if err := r.Client.Create(ctx, newWorkerPod); err != nil {
		return nil, fmt.Errorf("failed to create customCluster manager worker pod: %v", err)
	}
	return newWorkerPod, nil
}

// findManageWorkerPod locates the worker pod that has the given manageAction label and input OwnerReferences.
func (r *CustomClusterController) findManageWorkerPod(ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction) (*corev1.Pod, error) {
	labelSelector := client.MatchingLabels{ManageActionLabel: string(manageAction)}
	PodList := &corev1.PodList{}

	errGetWorker := r.Client.List(ctx, PodList, labelSelector)

	if errGetWorker != nil && !apierrors.IsNotFound(errGetWorker) {
		return nil, fmt.Errorf("failed to get worker pod when it should be deleted: %v", errGetWorker)
	}

	if errGetWorker == nil {
		// find the pod with an ownerRef that references this customCluster.
		for _, pod := range PodList.Items {
			// the current customCluster's worker has only one ownerRef.
			if pod.OwnerReferences[0].UID == customCluster.UID {
				return &pod, nil
			}
		}
	}
	return nil, nil
}

// getKubesprayImage takes a Kubernetes version string (in the format "vX.Y.Z")
// and returns the corresponding Kubespray image URL for that version.
// The function supports Kubernetes versions from 1.22.0 to 1.26.5.
// Kubespray v2.20.0 supports Kubernetes versions from 1.22.0 to 1.24.6,
// while Kubespray v2.22.1 supports Kubernetes versions from 1.24.0 to 1.26.5.
// This function returns one of these two Kubespray versions based on the input Kubernetes version.
func getKubesprayImage(ctx context.Context, kubeVersion string) string {
	log := ctrl.LoggerFrom(ctx)
	var kubesprayVersion string

	kubeVersion = strings.TrimPrefix(kubeVersion, "v")

	targetVersion, err := semver.NewVersion(kubeVersion)
	// should not happen, if we have validation on kubeVersion
	if err != nil {
		log.Error(err, "unexpected kube version", "targetVersion", targetVersion)
		return ""
	}

	midVersion, _ := semver.NewVersion("1.24.0")

	// Determine the Kubespray version based on the Kubernetes version.
	if targetVersion.Compare(*midVersion) >= 0 {
		kubesprayVersion = "v2.22.1"
	} else {
		kubesprayVersion = "v2.20.0"
	}

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

// fetchProvisionedClusterKubeConfig fetch provisioned cluster’s kubeConfig file, and create a secret named "provisionedClusterKubeConfigSecretPrefix + customCluster.name" with the data of kube-config file.
func (r *CustomClusterController) fetchProvisionedClusterKubeConfig(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine) error {
	remoteMachineSSHKey := customMachine.Spec.Master[0].SSHKey
	controlPlaneHost := customMachine.Spec.Master[0].PublicIP

	sshKeySecret, err := r.getSSHKeySecret(ctx, customMachine.Namespace, remoteMachineSSHKey.Name)
	if err != nil {
		return err
	}

	sshConfig, err := r.buildSSHClientConfig(sshKeySecret)
	if err != nil {
		return err
	}

	sftpClient, err := r.buildSFTPClient(controlPlaneHost+":22", sshConfig)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	kubeConfigData, err := r.fetchRemoteKubeConfig(sftpClient, ProvisionedKubeConfigPath)
	if err != nil {
		return err
	}

	err = r.createKubeConfigSecret(ctx, getKubeConfigSecretName(customCluster), customCluster.Namespace, kubeConfigData)
	if err != nil {
		return err
	}

	return nil
}

func (r *CustomClusterController) getSSHKeySecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	sshKeySecret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := r.Client.Get(ctx, key, sshKeySecret); err != nil {
		return nil, err
	}
	return sshKeySecret, nil
}

func (r *CustomClusterController) buildSSHClientConfig(sshKeySecret *corev1.Secret) (*ssh.ClientConfig, error) {
	sshPrivateKeyData, ok := sshKeySecret.Data["ssh-privatekey"]
	if !ok {
		return nil, fmt.Errorf("ssh-privatekey not found in secret")
	}

	signer, err := ssh.ParsePrivateKey(sshPrivateKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH private key: %v", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return sshConfig, nil
}

func (r *CustomClusterController) buildSFTPClient(addr string, sshConfig *ssh.ClientConfig) (*sftp.Client, error) {
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SFTP client: %v", err)
	}

	return sftpClient, nil
}

func (r *CustomClusterController) fetchRemoteKubeConfig(sftpClient *sftp.Client, path string) ([]byte, error) {
	remoteFile, err := sftpClient.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open remote file: %v", err)
	}
	defer remoteFile.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(remoteFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read remote file: %v", err)
	}

	return buf.Bytes(), nil
}

func (r *CustomClusterController) createKubeConfigSecret(ctx context.Context, name, namespace string, kubeConfigData []byte) error {
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"admin.conf": kubeConfigData,
		},
	}

	if err := r.Client.Create(ctx, newSecret); err != nil {
		return fmt.Errorf("failed to create new secret: %v", err)
	}

	return nil
}

func getKubeConfigSecretName(customCluster *v1alpha1.CustomCluster) string {
	return names.SimpleNameGenerator.GenerateName(customCluster.Name + "-kubeconfig-")
}
