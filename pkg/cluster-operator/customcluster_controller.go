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

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"kurator.dev/kurator/pkg/apis/infra/v1alpha1"
)

// CustomClusterController reconciles a CustomCluster object.
type CustomClusterController struct {
	client.Client
	APIReader client.Reader
	Scheme    *runtime.Scheme
}

type customClusterManageCMD string
type customClusterManageAction string

// NodeInfo represents the information of the node on VMs.
type NodeInfo struct {
	// NodeName, also called as HostName, is the unique identifier that distinguishes it from other nodes under the same cluster. kubespray uses the Hostname as the parameter to delete the node.
	NodeName  string
	PublicIP  string
	PrivateIP string
}

// ClusterInfo represents the information of the cluster on VMs.
type ClusterInfo struct {
	WorkerNodes []NodeInfo
	KubeVersion string
}

const (
	ClusterHostsName          = "cluster-hosts"
	ClusterConfigName         = "cluster-config"
	SecreteName               = "cluster-secret"
	ProvisionedKubeConfigPath = "/etc/kubernetes/admin.conf"

	ClusterKind       = "Cluster"
	CustomClusterKind = "CustomCluster"
	ManageActionLabel = "customcluster.kurator.dev/action"

	KubesprayCMDPrefix                                     = "ansible-playbook -i inventory/" + ClusterHostsName + " --private-key /root/.ssh/ssh-privatekey "
	CustomClusterInitAction      customClusterManageAction = "init"
	KubesprayInitCMD             customClusterManageCMD    = KubesprayCMDPrefix + "cluster.yml -vvv "
	CustomClusterTerminateAction customClusterManageAction = "terminate"
	KubesprayTerminateCMD        customClusterManageCMD    = KubesprayCMDPrefix + "reset.yml -vvv -e reset_confirmation=yes"

	CustomClusterScaleUpAction customClusterManageAction = "scale-up"
	KubesprayScaleUpCMD        customClusterManageCMD    = KubesprayCMDPrefix + "scale.yml -vvv "

	CustomClusterScaleDownAction customClusterManageAction = "scale-down"
	KubesprayScaleDownCMDPrefix  customClusterManageCMD    = KubesprayCMDPrefix + "remove-node.yml -vvv -e skip_confirmation=yes"

	CustomClusterUpgradeAction customClusterManageAction = "upgrade"
	KubesprayUpgradeCMDPrefix  customClusterManageCMD    = KubesprayCMDPrefix + "upgrade-cluster.yml -vvv "

	// CustomClusterFinalizer is the finalizer applied to crd.
	CustomClusterFinalizer = "customcluster.cluster.kurator.dev"
	// custom configmap finalizer requires at least one slash.
	CustomClusterConfigMapFinalizer = CustomClusterFinalizer + "/configmap"

	// KubeVersionPrefix is the prefix string of version of kubernetes
	KubeVersionPrefix = "kube_version: "
)

// SetupWithManager sets up the controller with the Manager.
func (r *CustomClusterController) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	c, err1 := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CustomCluster{}).
		WithOptions(options).
		Build(r)
	if err1 != nil {
		return fmt.Errorf("failed setting up with a controller manager: %v", err1)
	}

	if err := c.Watch(
		&source.Kind{Type: &corev1.Pod{}},
		handler.EnqueueRequestsFromMapFunc(r.WorkerToCustomClusterMapFunc),
	); err != nil {
		return fmt.Errorf("failed adding Watch for worker to controller manager: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &v1alpha1.CustomMachine{}},
		handler.EnqueueRequestsFromMapFunc(r.CustomMachineToCustomClusterMapFunc),
	); err != nil {
		return fmt.Errorf("failed adding Watch for CustomMachine to controller manager: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(r.ClusterToCustomClusterMapFunc),
	); err != nil {
		return fmt.Errorf("failed adding Watch for Clusters to controller manager: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &controlplanev1.KubeadmControlPlane{}},
		handler.EnqueueRequestsFromMapFunc(r.KcpToCustomClusterMapFunc),
	); err != nil {
		return fmt.Errorf("failed adding Watch for KubeadmControlPlan to controller manager: %v", err)
	}

	return nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *CustomClusterController) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx).WithValues("customCluster", req.NamespacedName)

	// Fetch the customCluster instance.
	customCluster := &v1alpha1.CustomCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, customCluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("customCluster does not exist")
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to find customCluster")
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// ensure customCluster status no nil
	if len(customCluster.Status.Phase) == 0 {
		customCluster.Status.Phase = v1alpha1.PendingPhase
	}

	// Fetch the Cluster instance.
	var clusterName string
	for _, owner := range customCluster.GetOwnerReferences() {
		if owner.Kind == ClusterKind {
			clusterName = owner.Name
			break
		}
	}
	if len(clusterName) == 0 {
		log.Info("failed to get cluster from customCluster.GetOwnerReferences")
		return ctrl.Result{}, nil
	}
	clusterKey := client.ObjectKey{
		Namespace: customCluster.GetNamespace(),
		Name:      clusterName,
	}
	cluster := &clusterv1.Cluster{}
	if err := r.Client.Get(ctx, clusterKey, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("cluster does not exist", "cluster", clusterKey)
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get cluster", "cluster", clusterKey)
		return ctrl.Result{}, err
	}

	// Fetch the CustomMachine instance.
	customMachinekey := client.ObjectKey{
		Namespace: customCluster.Spec.MachineRef.Namespace,
		Name:      customCluster.Spec.MachineRef.Name,
	}
	customMachine := &v1alpha1.CustomMachine{}
	if err := r.Client.Get(ctx, customMachinekey, customMachine); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("customMachine does not exist", "customMachine", customMachinekey)
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to find customMachine", "customMachine", customMachinekey)
		return ctrl.Result{}, err
	}

	// Fetch the KubeadmControlPlane instance.
	kcpKey := client.ObjectKey{
		Namespace: cluster.Spec.ControlPlaneRef.Namespace,
		Name:      cluster.Spec.ControlPlaneRef.Name,
	}
	kcp := &controlplanev1.KubeadmControlPlane{}
	if err := r.Client.Get(ctx, kcpKey, kcp); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("kcp does not exist", "kcp", kcpKey)
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get kcp", "kcp", kcpKey)
		return ctrl.Result{}, err
	}

	patchHelper, err1 := patch.NewHelper(customCluster, r.Client)
	if err1 != nil {
		return ctrl.Result{}, errors.Wrapf(err1, "failed to init patch helper for customCluster %s", req.NamespacedName)
	}

	defer func() {
		patchOpts := []patch.Option{
			patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
				v1alpha1.ReadyCondition,
				v1alpha1.ScaledUpCondition,
				v1alpha1.ScaledDownCondition,
				v1alpha1.TerminatedCondition,
			}},
		}

		if err := patchHelper.Patch(ctx, customCluster, patchOpts...); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch customCluster %s", req.NamespacedName)})
		}
	}()

	// Handle deletion reconciliation loop.
	if !cluster.DeletionTimestamp.IsZero() {
		phase := customCluster.Status.Phase
		if phase != v1alpha1.DeletingPhase {
			log.Info("phase changes", "prevPhase", customCluster.Status.Phase, "currentPhase", v1alpha1.DeletingPhase)
			customCluster.Status.Phase = v1alpha1.DeletingPhase
		}
		// Handle cluster deletion.
		return r.reconcileDelete(ctx, customCluster, customMachine, kcp)
	}

	// Handle normal loop.
	return r.reconcile(ctx, customCluster, customMachine, cluster, kcp)
}

// reconcile handles CustomCluster reconciliation.
func (r *CustomClusterController) reconcile(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, cluster *clusterv1.Cluster, kcp *controlplanev1.KubeadmControlPlane) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	phase := customCluster.Status.Phase

	// desiredClusterInfo contains information retrieved from configured CRDs such as "customMachine" and "kcp".
	desiredClusterInfo := getDesiredClusterInfo(customMachine, kcp)
	// desiredVersion is the one recorded in kcp.version.
	desiredVersion := desiredClusterInfo.KubeVersion

	// Handle cluster provision.
	if phase == v1alpha1.PendingPhase || phase == v1alpha1.ProvisionFailedPhase || phase == v1alpha1.ProvisioningPhase {
		return r.reconcileProvision(ctx, customCluster, customMachine, cluster, kcp)
	}

	// provisionedClusterInfo contains information retrieved from configmap that represent provisioned cluster.
	var provisionedClusterInfo *ClusterInfo
	// scaleUpWorkerNodes is the nodes where desiredCluster is more than provisionedCluster.
	var scaleUpWorkerNodes []NodeInfo
	// scaleUpWorkerNodes is the nodes where desiredCluster is less than provisionedCluster.
	var scaleDownWorkerNodes []NodeInfo
	// provisionedVersion is the one recorded in configmap cluster-config.data.kube_version.
	var provisionedVersion string
	if hasProvisionClusterInfo(phase) {
		var err error
		provisionedClusterInfo, err = r.getProvisionedClusterInfo(ctx, customCluster)
		if err != nil {
			log.Error(err, "failed to get provisioned cluster Info from configmap")
			return ctrl.Result{}, err
		}
		scaleUpWorkerNodes = findScaleUpWorkerNodes(provisionedClusterInfo.WorkerNodes, desiredClusterInfo.WorkerNodes)
		scaleDownWorkerNodes = findScaleDownWorkerNodes(provisionedClusterInfo.WorkerNodes, desiredClusterInfo.WorkerNodes)
		provisionedVersion = provisionedClusterInfo.KubeVersion
	}

	// Handle worker nodes scaling.
	// By comparing desiredClusterInfo.WorkerNodes and provisionedClusterInfo.WorkerNodes to decide whether to proceed reconcileScaleUp or reconcileScaleDown.
	if len(scaleUpWorkerNodes) != 0 {
		return r.reconcileScaleUp(ctx, customCluster, scaleUpWorkerNodes, kcp)
	}
	if len(scaleDownWorkerNodes) != 0 {
		return r.reconcileScaleDown(ctx, customCluster, customMachine, scaleDownWorkerNodes, kcp)
	}

	// Handle cluster upgrade.
	if desiredVersion != provisionedVersion {
		// If the desired version upgrade is not supported by Kubeadm, return directly.
		if !isKubeadmUpgradeSupported(provisionedVersion, desiredVersion) {
			log.Error(fmt.Errorf("skipping MINOR versions when upgrading is unsupported with kubeadm, you can not upgrade kubernetes version from %s to %s", provisionedVersion, desiredVersion), "")
			return ctrl.Result{}, nil
		}
		// Start reconcileUpgrade.
		return r.reconcileUpgrade(ctx, customCluster, desiredVersion)
	}

	return ctrl.Result{}, nil
}

// reconcileProvision handle cluster provision.
func (r *CustomClusterController) reconcileProvision(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, cluster *clusterv1.Cluster, kcp *controlplanev1.KubeadmControlPlane) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Create the configmaps that can be recognized by kubespray, which are derived from CRD parameters.
	clusterHosts, err1 := r.ensureClusterHostsCreated(ctx, customCluster, customMachine)
	if err1 != nil {
		log.Error(err1, "failed to update cluster-hosts configmap")
		return ctrl.Result{}, err1
	}
	clusterConfig, err2 := r.ensureClusterConfigCreated(ctx, customCluster, customCluster, cluster, kcp)
	if err2 != nil {
		log.Error(err2, "failed to update cluster-config configmap")
		return ctrl.Result{}, err2
	}

	// Create init worker pod to handle the provisioning process.
	initWorker, err3 := r.ensureWorkerPodCreated(ctx, customCluster, CustomClusterInitAction, KubesprayInitCMD, generateClusterHostsName(customCluster), generateClusterConfigName(customCluster), kcp.Spec.Version)
	if err3 != nil {
		conditions.MarkFalse(customCluster, v1alpha1.ReadyCondition, v1alpha1.FailedCreateInitWorker,
			clusterv1.ConditionSeverityWarning, "init worker is failed to create %s/%s", customCluster.Namespace, customCluster.Name)

		log.Error(err3, "failed to ensure that init WorkerPod is created ", "name", customCluster.Name, "namespace", customCluster.Namespace)
		return ctrl.Result{}, err3
	}

	// Ensure that the object's finalizer and owner reference are appropriately set.
	if err := r.ensureFinalizerAndOwnerRef(ctx, clusterHosts, clusterConfig, customCluster, customMachine, kcp); err != nil {
		log.Error(err, "failed to set finalizer or ownerRefs")
		return ctrl.Result{}, err
	}

	if customCluster.Status.Phase != v1alpha1.ProvisioningPhase {
		log.Info("phase changes", "prevPhase", customCluster.Status.Phase, "currentPhase", v1alpha1.ProvisioningPhase)
		customCluster.Status.Phase = v1alpha1.ProvisioningPhase
	}

	// The provisioning process will be successfully completed if the init worker is finished successfully.
	if initWorker.Status.Phase == corev1.PodSucceeded {
		if err := r.fetchProvisionedClusterKubeConfig(ctx, customCluster, customMachine); err != nil {
			log.Error(err, "failed to fetch provisioned cluster kubeConfig")
			conditions.MarkFalse(customCluster, v1alpha1.ObtainedKubeConfigCondition, v1alpha1.FailedFetchKubeConfigReason,
				clusterv1.ConditionSeverityWarning, "failed to fetch provisioned cluster KubeConfig %s/%s", customCluster.Namespace, customCluster.Name)
			return ctrl.Result{}, err
		}
		conditions.MarkTrue(customCluster, v1alpha1.ObtainedKubeConfigCondition)
		log.Info("phase changes", "prevPhase", customCluster.Status.Phase, "currentPhase", v1alpha1.ProvisionedPhase)
		customCluster.Status.Phase = v1alpha1.ProvisionedPhase
		conditions.MarkTrue(customCluster, v1alpha1.ReadyCondition)
		return ctrl.Result{}, nil
	}
	if initWorker.Status.Phase == corev1.PodFailed {
		log.Info("phase changes", "prevPhase", customCluster.Status.Phase, "currentPhase", v1alpha1.ProvisionFailedPhase)
		customCluster.Status.Phase = v1alpha1.ProvisionFailedPhase
		conditions.MarkFalse(customCluster, v1alpha1.ReadyCondition, v1alpha1.InitWorkerRunFailedReason,
			clusterv1.ConditionSeverityWarning, "init worker run failed %s/%s", customCluster.Namespace, customCluster.Name)

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// reconcileDelete handle cluster deletion.
func (r *CustomClusterController) reconcileDelete(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, kcp *controlplanev1.KubeadmControlPlane) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Delete the manager worker pods first if there are still any running.
	if err := r.deleteWorkerPods(ctx, customCluster); err != nil {
		log.Error(err, "failed to delete worker pods", "name", customCluster.Name, "namespace", customCluster.Namespace)
		return ctrl.Result{}, err
	}

	// Create the termination worker to handle the cluster deletion.
	terminateWorker, err1 := r.ensureWorkerPodCreated(ctx, customCluster, CustomClusterTerminateAction, KubesprayTerminateCMD, generateClusterHostsName(customCluster), generateClusterConfigName(customCluster), kcp.Spec.Version)
	if err1 != nil {
		conditions.MarkFalse(customCluster, v1alpha1.TerminatedCondition, v1alpha1.FailedCreateTerminateWorker,
			clusterv1.ConditionSeverityWarning, "terminate worker is failed to create %s/%s.", customCluster.Namespace, customCluster.Name)
		log.Error(err1, "failed to create terminate worker", "name", customCluster.Name, "namespace", customCluster.Namespace)
		return ctrl.Result{}, err1
	}

	// After k8s cluster has been deleted successful, we need delete the related CRD.
	if terminateWorker.Status.Phase == corev1.PodSucceeded {
		log.Info("terminating worker was completed successfully, delete the related CRD")
		if err := r.deleteResource(ctx, customCluster, customMachine, kcp); err != nil {
			log.Error(err, "failed to delete resource", "name", customCluster.Name, "namespace", customCluster.Namespace)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if terminateWorker.Status.Phase == corev1.PodFailed {
		log.Info("phase changes", "prevPhase", customCluster.Status.Phase, "currentPhase", v1alpha1.UnknownPhase)
		customCluster.Status.Phase = v1alpha1.UnknownPhase
		conditions.MarkFalse(customCluster, v1alpha1.TerminatedCondition, v1alpha1.TerminateWorkerRunFailedReason,
			clusterv1.ConditionSeverityWarning, "terminate worker run failed %s/%s.", customCluster.Namespace, customCluster.Name)

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// deleteWorkerPods delete all the manage worker pods, including those for initialization, scaling up, scaling down, and other related tasks.
func (r *CustomClusterController) deleteWorkerPods(ctx context.Context, customCluster *v1alpha1.CustomCluster) error {
	log := ctrl.LoggerFrom(ctx)

	// Delete the init worker.
	if err := r.ensureWorkerPodDeleted(ctx, customCluster, CustomClusterInitAction); err != nil {
		log.Error(err, "failed to delete init worker", "name", customCluster.Name, "namespace", customCluster.Namespace)
		return err
	}

	// Delete the scale up worker.
	if err := r.ensureWorkerPodDeleted(ctx, customCluster, CustomClusterScaleUpAction); err != nil {
		log.Error(err, "failed to delete scale up worker", "name", customCluster.Name, "namespace", customCluster.Namespace)
		return err
	}

	// Delete the scale down worker.
	if err := r.ensureWorkerPodDeleted(ctx, customCluster, CustomClusterScaleDownAction); err != nil {
		log.Error(err, "failed to delete scale down worker", "name", customCluster.Name, "namespace", customCluster.Namespace)
		return err
	}

	return nil
}

// deleteResource delete resources associated with customCluster, including configmaps, customMachines, customClusters and so on.
func (r *CustomClusterController) deleteResource(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, kcp *controlplanev1.KubeadmControlPlane) error {
	log := ctrl.LoggerFrom(ctx)

	// Delete the configmap cluster-hosts.
	if err := r.ensureConfigMapDeleted(ctx, generateClusterHostsKey(customCluster)); err != nil {
		log.Error(err, "failed to ensure that configmap is deleted", "configmap", generateClusterHostsKey(customCluster))
		return err
	}
	// Delete the configmap cluster-config.
	if err := r.ensureConfigMapDeleted(ctx, generateClusterConfigKey(customCluster)); err != nil {
		log.Error(err, "failed to ensure that configmap is deleted", "configmap", generateClusterConfigKey(customCluster))
		return err
	}

	// Remove finalizer of customMachine.
	controllerutil.RemoveFinalizer(customMachine, CustomClusterFinalizer)
	if err := r.Client.Update(ctx, customMachine); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to remove finalizer of customMachine", "name", customMachine.Name, "namespace", customMachine.Namespace)
		return err
	}

	// Remove finalizer of kcp.
	controllerutil.RemoveFinalizer(kcp, CustomClusterFinalizer)
	if err := r.Client.Update(ctx, kcp); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to remove finalizer of kcp", "name", kcp.Name, "namespace", kcp.Namespace)
		return err
	}

	// Remove finalizer of customCluster. After this, cluster will be deleted completely.
	controllerutil.RemoveFinalizer(customCluster, CustomClusterFinalizer)
	if err := r.Client.Update(ctx, customCluster); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to remove finalizer of customCluster", "name", customCluster.Name, "namespace", customCluster.Namespace)
		return err
	}

	return nil
}

// ensureFinalizerAndOwnerRef ensure related resource's finalizer and ownerRef is ready.
func (r *CustomClusterController) ensureFinalizerAndOwnerRef(ctx context.Context, clusterHosts *corev1.ConfigMap, clusterConfig *corev1.ConfigMap, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, kcp *controlplanev1.KubeadmControlPlane) error {
	controllerutil.AddFinalizer(customCluster, CustomClusterFinalizer)
	controllerutil.AddFinalizer(customMachine, CustomClusterFinalizer)
	controllerutil.AddFinalizer(kcp, CustomClusterFinalizer)
	controllerutil.AddFinalizer(clusterHosts, CustomClusterConfigMapFinalizer)
	controllerutil.AddFinalizer(clusterConfig, CustomClusterConfigMapFinalizer)

	ownerRefs := generateOwnerRefFromCustomCluster(customCluster)
	customMachine.OwnerReferences = capiutil.EnsureOwnerRef(customMachine.OwnerReferences, ownerRefs)
	clusterHosts.OwnerReferences = capiutil.EnsureOwnerRef(clusterHosts.OwnerReferences, ownerRefs)
	clusterConfig.OwnerReferences = capiutil.EnsureOwnerRef(clusterConfig.OwnerReferences, ownerRefs)

	if err := r.Client.Update(ctx, customMachine); err != nil {
		return fmt.Errorf("failed to set finalizer or ownerRef of customMachine: %v", err)
	}

	if err := r.Client.Update(ctx, clusterHosts); err != nil {
		return fmt.Errorf("failed to set finalizer or ownerRef of clusterHosts: %v", err)
	}

	if err := r.Client.Update(ctx, clusterConfig); err != nil {
		return fmt.Errorf("failed to set finalizer or ownerRef of clusterConfig: %v", err)
	}

	if err := r.Client.Update(ctx, kcp); err != nil {
		return fmt.Errorf("failed to set finalizer of kcp: %v", err)
	}

	return nil
}

func (r *CustomClusterController) WorkerToCustomClusterMapFunc(o client.Object) []ctrl.Request {
	c, ok := o.(*corev1.Pod)
	if !ok {
		panic(fmt.Sprintf("Expected a pod but got a %T", o))
	}
	for _, owner := range c.GetOwnerReferences() {
		if owner.Kind == CustomClusterKind {
			return []ctrl.Request{{NamespacedName: client.ObjectKey{Namespace: c.Namespace, Name: owner.Name}}}
		}
	}
	return nil
}

func (r *CustomClusterController) CustomMachineToCustomClusterMapFunc(o client.Object) []ctrl.Request {
	c, ok := o.(*v1alpha1.CustomMachine)
	if !ok {
		panic(fmt.Sprintf("Expected a CustomMachine but got a %T", o))
	}

	var result []ctrl.Request
	for _, owner := range c.GetOwnerReferences() {
		if owner.Kind == CustomClusterKind {
			name := client.ObjectKey{Namespace: c.GetNamespace(), Name: owner.Name}
			result = append(result, ctrl.Request{NamespacedName: name})
			break
		}
	}
	return result
}

func (r *CustomClusterController) ClusterToCustomClusterMapFunc(o client.Object) []ctrl.Request {
	c, ok := o.(*clusterv1.Cluster)
	if !ok {
		panic(fmt.Sprintf("Expected a Cluster but got a %T", o))
	}
	infrastructureRef := c.Spec.InfrastructureRef
	if infrastructureRef != nil && infrastructureRef.Kind == CustomClusterKind {
		return []ctrl.Request{{NamespacedName: client.ObjectKey{Namespace: infrastructureRef.Namespace, Name: infrastructureRef.Name}}}
	}
	return nil
}

func (r *CustomClusterController) KcpToCustomClusterMapFunc(o client.Object) []ctrl.Request {
	c, ok := o.(*controlplanev1.KubeadmControlPlane)
	if !ok {
		panic(fmt.Sprintf("Expected a KubeadmControlPlane but got a %T", o))
	}
	var result []ctrl.Request

	// Find the cluster from kcp.
	clusterKey := client.ObjectKey{}
	for _, owner := range c.GetOwnerReferences() {
		if owner.Kind == ClusterKind {
			clusterKey = client.ObjectKey{Namespace: c.GetNamespace(), Name: owner.Name}
			break
		}
	}
	ownerCluster := &clusterv1.Cluster{}
	if err := r.Client.Get(context.TODO(), clusterKey, ownerCluster); err != nil {
		return nil
	}

	// Find the customCluster from cluster.
	infrastructureRef := ownerCluster.Spec.InfrastructureRef
	if infrastructureRef != nil && infrastructureRef.Kind == CustomClusterKind {
		return []ctrl.Request{{NamespacedName: client.ObjectKey{Namespace: infrastructureRef.Namespace, Name: infrastructureRef.Name}}}
	}
	return result
}
