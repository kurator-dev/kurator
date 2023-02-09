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
	"fmt"
	"strings"
	"text/template"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
)

// CustomClusterController reconciles a CustomCluster object.
type CustomClusterController struct {
	client.Client
	APIReader client.Reader
	Scheme    *runtime.Scheme
}

type customClusterManageCMD string
type customClusterManageAction string

const (
	RequeueAfter = time.Second * 5

	ClusterHostsName  = "cluster-hosts"
	ClusterConfigName = "cluster-config"
	SecreteName       = "cluster-secret"

	ClusterKind       = "Cluster"
	CustomClusterKind = "CustomCluster"

	CustomClusterInitAction      customClusterManageAction = "init"
	KubesprayInitCMD             customClusterManageCMD    = "ansible-playbook -i inventory/" + ClusterHostsName + " --private-key /root/.ssh/ssh-privatekey cluster.yml -vvv"
	CustomClusterTerminateAction customClusterManageAction = "terminate"
	KubesprayTerminateCMD        customClusterManageCMD    = "ansible-playbook -e reset_confirmation=yes -i inventory/" + ClusterHostsName + " --private-key /root/.ssh/ssh-privatekey reset.yml -vvv"

	// TODO: support custom this in CustomCluster/CustomMachine
	DefaultKubesprayImage = "quay.io/kubespray/kubespray:v2.20.0"

	// CustomClusterFinalizer is the finalizer applied to crd.
	CustomClusterFinalizer = "customcluster.cluster.kurator.dev"
	// custom configmap finalizer requires at least one slash.
	CustomClusterConfigMapFinalizer = CustomClusterFinalizer + "/configmap"
)

// SetupWithManager sets up the controller with the Manager.
func (r *CustomClusterController) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CustomCluster{}).
		WithOptions(options).
		Build(r)
	if err != nil {
		return fmt.Errorf("failed setting up with a controller manager: %v", err)
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
func (r *CustomClusterController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the customCluster instance.
	customCluster := &v1alpha1.CustomCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, customCluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("customCluster does not exist", "customCluster", req)
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to find customCluster", "customCluster", req)
		// Error reading the object - requeue the request.
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
	}
	log = log.WithValues("customCluster", klog.KObj(customCluster))
	ctx = ctrl.LoggerInto(ctx, log)
	// ensure customCluster status no nil
	if len(customCluster.Status.Phase) == 0 {
		customCluster.Status.Phase = v1alpha1.PendingPhase
		if err := r.Status().Update(ctx, customCluster); err != nil {
			log.Error(err, "failed to update customCluster status", "customCluster", req)
			return ctrl.Result{RequeueAfter: RequeueAfter}, err
		}
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
		log.Info("failed to get cluster from customCluster.GetOwnerReferences", "customCluster", req)
		return ctrl.Result{RequeueAfter: RequeueAfter}, nil
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
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
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
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
	}

	return r.reconcile(ctx, customCluster, customMachine, cluster)
}

// reconcile handles CustomCluster reconciliation.
func (r *CustomClusterController) reconcile(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, cluster *clusterv1.Cluster) (ctrl.Result, error) {
	phase := customCluster.Status.Phase

	// If upstream cluster at pre-delete, the customCluster need to be deleted.
	if !cluster.DeletionTimestamp.IsZero() {
		// If customCluster is already in phase Deleting, the controller will check the terminating worker to handle Deleting process.
		if phase == v1alpha1.DeletingPhase {
			return r.reconcileHandleDeleting(ctx, customCluster, customMachine)
		}
		// If customCluster is not in Deleting, the controller should terminate the Vms cluster by create a terminating worker.
		return r.reconcileVMsTerminate(ctx, customCluster)
	}

	// CustomCluster in phase nil or ProvisionFailed will try to enter Provisioning phase by creating an init worker successfully.
	if phase == v1alpha1.PendingPhase || phase == v1alpha1.ProvisionFailedPhase {
		return r.reconcileCustomClusterInit(ctx, customCluster, customMachine, cluster)
	}

	// If customCluster is in phase Provisioning, the controller will handle Provisioning process by checking the init worker status.
	if phase == v1alpha1.ProvisioningPhase {
		return r.reconcileHandleProvisioning(ctx, customCluster, customMachine, cluster)
	}

	return ctrl.Result{}, nil
}

// reconcileHandleProvisioning determine whether customCluster enter Provisioned phase or ProvisionFailed phase when current phase is Provisioning.
func (r *CustomClusterController) reconcileHandleProvisioning(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, cluster *clusterv1.Cluster) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	initWorker := &corev1.Pod{}
	initWorkerKey := generateWorkerKey(customCluster, CustomClusterInitAction)
	if err := r.Client.Get(ctx, initWorkerKey, initWorker); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("init worker does not exist, turn to reconcileCustomClusterInit to create a new one", "worker", initWorkerKey)
			return r.reconcileCustomClusterInit(ctx, customCluster, customMachine, cluster)
		}
		log.Error(err, "failed to get init worker", "worker", initWorkerKey)
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
	}

	if initWorker.Status.Phase == corev1.PodSucceeded {
		customCluster.Status.Phase = v1alpha1.ProvisionedPhase
		if err := r.Status().Update(ctx, customCluster); err != nil {
			log.Error(err, "failed to update customCluster status", "customCluster", customCluster.Name)
			return ctrl.Result{RequeueAfter: RequeueAfter}, err
		}
		log.Info("customCluster's phase changes from Provisioning to Provisioned")
		return ctrl.Result{}, nil
	}

	if initWorker.Status.Phase == corev1.PodFailed {
		customCluster.Status.Phase = v1alpha1.ProvisionFailedPhase
		if err := r.Status().Update(ctx, customCluster); err != nil {
			log.Error(err, "failed to update customCluster status", "customCluster", customCluster.Name)
			return ctrl.Result{RequeueAfter: RequeueAfter}, err
		}
		log.Info("customCluster's phase changes from Provisioning to ProvisionFailed")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// reconcileHandleDeleting determine whether customCluster go to reconcileDeleteResource or enter DeletingFailed phase when current phase is Deleting.
func (r *CustomClusterController) reconcileHandleDeleting(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	terminateWorker := &corev1.Pod{}
	terminateWorkerKey := generateWorkerKey(customCluster, CustomClusterTerminateAction)

	if err := r.Client.Get(ctx, terminateWorkerKey, terminateWorker); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("terminate worker does not exist, turn to reconcileVMsTerminate to create a new one", "worker", terminateWorkerKey)
			return r.reconcileVMsTerminate(ctx, customCluster)
		}
		log.Error(err, "failed to get terminate worker. maybe it has been deleted", "worker", terminateWorkerKey)
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
	}

	if terminateWorker.Status.Phase == corev1.PodSucceeded {
		log.Info("terminating worker was completed successfully, then we need delete the related CRD")
		// After k8s cluster on VMs has been reset successful, we need delete the related CRD.
		return r.reconcileDeleteResource(ctx, customCluster, customMachine)
	}

	if terminateWorker.Status.Phase == corev1.PodFailed {
		customCluster.Status.Phase = v1alpha1.DeletingFailedPhase
		if err := r.Status().Update(ctx, customCluster); err != nil {
			log.Error(err, "failed to update customCluster status", "customCluster", customCluster.Name)
			return ctrl.Result{RequeueAfter: RequeueAfter}, err
		}
		log.Info("customCluster's phase changes from Deleting to DeletingFailed")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// reconcileVMsTerminate uninstall the k8s cluster on VMs.
func (r *CustomClusterController) reconcileVMsTerminate(ctx context.Context, customCluster *v1alpha1.CustomCluster) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Delete init worker.
	initWorker := &corev1.Pod{}
	initWorkerKey := generateWorkerKey(customCluster, CustomClusterInitAction)
	errGetWorker := r.Client.Get(ctx, initWorkerKey, initWorker)
	// errGetWorker can be divided into three situation: isNotFound; not isNotFound; nil.
	if apierrors.IsNotFound(errGetWorker) {
		log.Info("init worker already deleted, no action", "worker", initWorkerKey)
	} else if errGetWorker != nil && !apierrors.IsNotFound(errGetWorker) {
		log.Error(errGetWorker, "failed to get init worker when it should be deleted", "worker", initWorkerKey)
		return ctrl.Result{RequeueAfter: RequeueAfter}, errGetWorker
	} else if errGetWorker == nil {
		if err := r.Client.Delete(ctx, initWorker); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "failed to delete init worker", "worker", initWorkerKey)
			return ctrl.Result{RequeueAfter: RequeueAfter}, err
		}
		log.Info("init worker was deleted successfully", "worker", initWorkerKey)
	}

	// Check if terminate-worker already exist. if not, create it.
	terminateWorkerKey := generateWorkerKey(customCluster, CustomClusterTerminateAction)
	terminateWorkerPod := &corev1.Pod{}
	if err := r.Client.Get(ctx, terminateWorkerKey, terminateWorkerPod); err != nil {
		if apierrors.IsNotFound(err) {
			terminateClusterPod := r.generateClusterManageWorker(customCluster, CustomClusterTerminateAction, KubesprayTerminateCMD)
			terminateClusterPod.OwnerReferences = []metav1.OwnerReference{generateOwnerRefFromCustomCluster(customCluster)}
			if err1 := r.Client.Create(ctx, terminateClusterPod); err1 != nil {
				log.Error(err1, "failed to create customCluster terminate worker", "worker", terminateWorkerKey)
				return ctrl.Result{RequeueAfter: RequeueAfter}, err1
			}
		}
		log.Error(err, "failed to get terminate worker", "worker", terminateWorkerKey)
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
	}

	customCluster.Status.Phase = v1alpha1.DeletingPhase
	if err := r.Status().Update(ctx, customCluster); err != nil {
		log.Error(err, "failed to update customCluster status", "customCluster", customCluster.Name)
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
	}
	log.Info("customCluster's phase changes to Deleting")

	return ctrl.Result{}, nil
}

// reconcileDeleteResource delete resource related to customCluster: configmap, pod, customMachine, customCluster.
func (r *CustomClusterController) reconcileDeleteResource(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Remove finalizer of clusterHosts.
	clusterHostsKey := generateClusterHostsKey(customCluster)
	clusterHosts := &corev1.ConfigMap{}
	errGetClusterHosts := r.Client.Get(ctx, clusterHostsKey, clusterHosts)
	// errGetClusterHosts can be divided into three situation: isNotFound; not isNotFound; nil.
	if apierrors.IsNotFound(errGetClusterHosts) {
		log.Info("clusterHosts already deleted, no action", "clusterHosts", clusterHostsKey)
	} else if errGetClusterHosts != nil && !apierrors.IsNotFound(errGetClusterHosts) {
		log.Error(errGetClusterHosts, "failed to get clusterHosts when it should be deleted", "clusterHosts", clusterHostsKey)
		return ctrl.Result{RequeueAfter: RequeueAfter}, errGetClusterHosts
	} else if errGetClusterHosts == nil {
		controllerutil.RemoveFinalizer(clusterHosts, CustomClusterConfigMapFinalizer)
		if err := r.Client.Update(ctx, clusterHosts); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "failed to remove finalizer of clusterHosts when it should be deleted", "clusterHosts", clusterHostsKey)
			return ctrl.Result{RequeueAfter: RequeueAfter}, err
		}
		log.Info("finalizer of clusterHosts was removed successfully", "clusterHosts", clusterHostsKey)
	}

	// Remove finalizer of clusterConfig.
	clusterConfigKey := generateClusterConfigKey(customCluster)
	clusterConfig := &corev1.ConfigMap{}
	errGetClusterConfig := r.Client.Get(ctx, clusterConfigKey, clusterConfig)
	// errGetClusterConfig can be divided into three situation: isNotFound; not isNotFound; nil.
	if apierrors.IsNotFound(errGetClusterConfig) {
		log.Info("clusterConfig already deleted, no action", "clusterConfig", clusterConfigKey)
	} else if errGetClusterConfig != nil && !apierrors.IsNotFound(errGetClusterConfig) {
		log.Error(errGetClusterConfig, "failed to get clusterConfig when it should be deleted", "clusterConfig", clusterConfigKey)
		return ctrl.Result{RequeueAfter: RequeueAfter}, errGetClusterConfig
	} else if errGetClusterConfig == nil {
		controllerutil.RemoveFinalizer(clusterConfig, CustomClusterConfigMapFinalizer)
		if err := r.Client.Update(ctx, clusterConfig); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "failed to remove finalizer of clusterConfig when it should be deleted", "clusterConfig", clusterConfigKey)
			return ctrl.Result{RequeueAfter: RequeueAfter}, err
		}
		log.Info("finalizer of clusterConfig was removed successfully", "clusterConfig", clusterConfigKey)
	}

	// Remove finalizer of customMachine.
	controllerutil.RemoveFinalizer(customMachine, CustomClusterFinalizer)
	if err := r.Client.Update(ctx, customMachine); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to remove finalizer of customMachine", "customMachine", customMachine.Name)
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
	}

	// Remove finalizer of customCluster. After this, cluster will be deleted completely.
	controllerutil.RemoveFinalizer(customCluster, CustomClusterFinalizer)
	if err := r.Client.Update(ctx, customCluster); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to remove finalizer of customCluster", "customCluster", customCluster.Name)
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
	}

	return ctrl.Result{}, nil
}

type RelatedResource struct {
	clusterHosts  *corev1.ConfigMap
	clusterConfig *corev1.ConfigMap
	customCluster *v1alpha1.CustomCluster
	customMachine *v1alpha1.CustomMachine
}

// reconcileCustomClusterInit create an init worker for installing cluster on VMs.
func (r *CustomClusterController) reconcileCustomClusterInit(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, cluster *clusterv1.Cluster) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

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
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
	}

	var clusterHost *corev1.ConfigMap
	var errorHost error
	if clusterHost, errorHost = r.updateClusterHosts(ctx, customCluster, customMachine); errorHost != nil {
		log.Error(errorHost, "failed to update cluster-hosts configMap")
		return ctrl.Result{RequeueAfter: RequeueAfter}, errorHost
	}

	var clusterConfig *corev1.ConfigMap
	var errorConfig error
	if clusterConfig, errorConfig = r.updateClusterConfig(ctx, customCluster, customCluster, cluster, kcp); errorConfig != nil {
		log.Error(errorConfig, "failed to update cluster-config configMap")
		return ctrl.Result{RequeueAfter: RequeueAfter}, errorConfig
	}

	// Check if init worker already exist. If not, create it.
	initWorkerKey := generateWorkerKey(customCluster, CustomClusterInitAction)
	initWorker := &corev1.Pod{}
	if err := r.Client.Get(ctx, initWorkerKey, initWorker); err != nil {
		if apierrors.IsNotFound(err) {
			initClusterPod := r.generateClusterManageWorker(customCluster, CustomClusterInitAction, KubesprayInitCMD)
			initClusterPod.OwnerReferences = []metav1.OwnerReference{generateOwnerRefFromCustomCluster(customCluster)}
			if err1 := r.Client.Create(ctx, initClusterPod); err1 != nil {
				log.Error(err1, "failed to create customCluster init worker", "initWorker", initWorkerKey)
				return ctrl.Result{RequeueAfter: RequeueAfter}, err1
			}
		}
		log.Error(err, "failed to get init worker", "initWorker", initWorkerKey)
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
	}

	initRelatedResource := &RelatedResource{
		clusterHosts:  clusterHost,
		clusterConfig: clusterConfig,
		customCluster: customCluster,
		customMachine: customMachine,
	}
	// When all related object is ready, we need ensure object's finalizer and ownerRef is set appropriately.
	if err := r.ensureFinalizerAndOwnerRef(ctx, initRelatedResource); err != nil {
		log.Error(err, "failed to set finalizer or ownerRefs")
		return ctrl.Result{RequeueAfter: RequeueAfter}, err
	}

	customCluster.Status.Phase = v1alpha1.ProvisioningPhase
	if err1 := r.Status().Update(ctx, customCluster); err1 != nil {
		log.Error(err1, "failed to update customCluster status", "customCluster", customCluster.Name)
		return ctrl.Result{RequeueAfter: RequeueAfter}, err1
	}
	log.Info("customCluster's phase changes to Provisioning")

	return ctrl.Result{}, nil
}

// ensureFinalizerAndOwnerRef ensure every related resource's finalizer and ownerRef is ready.
func (r *CustomClusterController) ensureFinalizerAndOwnerRef(ctx context.Context, res *RelatedResource) error {
	controllerutil.AddFinalizer(res.customCluster, CustomClusterFinalizer)
	controllerutil.AddFinalizer(res.customMachine, CustomClusterFinalizer)
	controllerutil.AddFinalizer(res.clusterHosts, CustomClusterConfigMapFinalizer)
	controllerutil.AddFinalizer(res.clusterConfig, CustomClusterConfigMapFinalizer)

	ownerRefs := generateOwnerRefFromCustomCluster(res.customCluster)
	res.customMachine.OwnerReferences = []metav1.OwnerReference{ownerRefs}
	res.clusterHosts.OwnerReferences = []metav1.OwnerReference{ownerRefs}
	res.clusterConfig.OwnerReferences = []metav1.OwnerReference{ownerRefs}

	if err := r.Client.Update(ctx, res.customMachine); err != nil {
		return fmt.Errorf("failed to set finalizer or ownerRef of customMachine: %v", err)
	}

	if err := r.Client.Update(ctx, res.clusterHosts); err != nil {
		return fmt.Errorf("failed to set finalizer or ownerRef of clusterHosts: %v", err)
	}

	if err := r.Client.Update(ctx, res.clusterConfig); err != nil {
		return fmt.Errorf("failed to set finalizer or ownerRef of clusterConfig: %v", err)
	}

	if err := r.Client.Update(ctx, res.customCluster); err != nil {
		return fmt.Errorf("failed to set finalizer or ownerRef of customCluster: %v", err)
	}

	return nil
}

// generateClusterManageWorker generate a kubespray init cluster pod from configMap.
func (r *CustomClusterController) generateClusterManageWorker(customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction, manageCMD customClusterManageCMD) *corev1.Pod {
	podName := customCluster.Name + "-" + string(manageAction)
	namespace := customCluster.Namespace
	defaultMode := int32(0o600)

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
					Image:   DefaultKubesprayImage,
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
								Name: customCluster.Name + "-" + ClusterHostsName,
							},
						},
					},
				},
				{
					Name: ClusterConfigName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: customCluster.Name + "-" + ClusterConfigName,
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
	PodCIDR     string
	// CNIType is the CNI plugin of the cluster on VMs. The default plugin is calico and can be ["calico", "cilium", "canal", "flannel"]
	CNIType string
	// TODO: support other kubernetes configs
}

func GetConfigContent(c *clusterv1.Cluster, kcp *controlplanev1.KubeadmControlPlane, cc *v1alpha1.CustomCluster) *ConfigTemplateContent {
	// Add kubespray init config here
	configContent := &ConfigTemplateContent{
		PodCIDR:     c.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		KubeVersion: kcp.Spec.Version,
		CNIType:     cc.Spec.CNI.Type,
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

func (r *CustomClusterController) CreateClusterHosts(ctx context.Context, customMachine *v1alpha1.CustomMachine, customCluster *v1alpha1.CustomCluster) (*corev1.ConfigMap, error) {
	hostsContent := GetHostsContent(customMachine)
	hostData := &strings.Builder{}

	// todo: split this to a separated file
	tmpl := template.Must(template.New("").Parse(`
[all]
{{ range $v := .NodeAndIP }}
{{ $v }}
{{ end }}
[kube_control_plane]
{{ range $v := .MasterName }}
{{ $v }}
{{ end }}
[etcd]
{{- range $v := .EtcdNodeName }}
{{ $v }}
{{ end }}
[kube_node]
{{- range $v := .NodeName }}
{{ $v }}
{{ end }}
[k8s-cluster:children]
kube_node
kube_control_plane
`))

	if err := tmpl.Execute(hostData, hostsContent); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s-%s", customCluster.Name, ClusterHostsName)
	namespace := customCluster.Namespace

	return r.CreateConfigMapWithTemplate(ctx, name, namespace, ClusterHostsName, hostData.String())
}

func (r *CustomClusterController) CreateClusterConfig(ctx context.Context, c *clusterv1.Cluster, kcp *controlplanev1.KubeadmControlPlane, cc *v1alpha1.CustomCluster) (*corev1.ConfigMap, error) {
	configContent := GetConfigContent(c, kcp, cc)
	configData := &strings.Builder{}

	// todo: split this to a separated file
	tmpl := template.Must(template.New("").Parse(`
kube_version: {{ .KubeVersion}}
download_run_once: true
download_container: false
download_localhost: true
# network
kube_pods_subnet: {{ .PodCIDR }}
kube_network_plugin: {{ .CNIType }}
kubeconfig_localhost: true
artifacts_dir: "/root/.kube"
`))

	if err := tmpl.Execute(configData, configContent); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s-%s", cc.Name, ClusterConfigName)
	namespace := cc.Namespace

	return r.CreateConfigMapWithTemplate(ctx, name, namespace, ClusterConfigName, configData.String())
}

// updateClusterHosts. If cluster-hosts configmap does not exist, create it.
func (r *CustomClusterController) updateClusterHosts(ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine) (*corev1.ConfigMap, error) {
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

// updateClusterConfig. If cluster-config configmap does not exist, create it.
func (r *CustomClusterController) updateClusterConfig(ctx context.Context, customCluster *v1alpha1.CustomCluster, cc *v1alpha1.CustomCluster, cluster *clusterv1.Cluster, kcp *controlplanev1.KubeadmControlPlane) (*corev1.ConfigMap, error) {
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

func (r *CustomClusterController) WorkerToCustomClusterMapFunc(o client.Object) []ctrl.Request {
	c, ok := o.(*corev1.Pod)
	if !ok {
		panic(fmt.Sprintf("Expected a Cluster but got a %T", o))
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

func generateWorkerKey(customCluster *v1alpha1.CustomCluster, action customClusterManageAction) client.ObjectKey {
	return client.ObjectKey{
		Namespace: customCluster.Namespace,
		Name:      customCluster.Name + "-" + string(action),
	}
}

func generateClusterHostsKey(customCluster *v1alpha1.CustomCluster) client.ObjectKey {
	return client.ObjectKey{
		Namespace: customCluster.Namespace,
		Name:      customCluster.Name + "-" + ClusterHostsName,
	}
}

func generateClusterConfigKey(customCluster *v1alpha1.CustomCluster) client.ObjectKey {
	return client.ObjectKey{
		Namespace: customCluster.Namespace,
		Name:      customCluster.Name + "-" + ClusterConfigName,
	}
}

func generateOwnerRefFromCustomCluster(customCluster *v1alpha1.CustomCluster) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: customCluster.APIVersion,
		Kind:       customCluster.Kind,
		Name:       customCluster.Name,
		UID:        customCluster.UID,
	}
}
