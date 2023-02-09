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

package infra

import (
	"context"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1 "kurator.dev/kurator/pkg/apis/infra/v1alpha1"
	"kurator.dev/kurator/pkg/util/names"
)

const (
	// ClusterFinalizer allows ClusterController to clean up associated resources before removing it from apiserver.
	clusterFinalizer = "cluster.infra.kurator.dev"
)

// ClusterController reconciles a Cluster object
type ClusterController struct {
	client.Client
	Scheme        *runtime.Scheme
	NameGenerator names.Generator
	PollInterval  time.Duration
}

func (r *ClusterController) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctxLogger := log.FromContext(ctx)

	infraCluster := &infrav1.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, infraCluster); err != nil {
		if apiserrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrapf(err, "failed to get infra Cluster %s", req.NamespacedName)
	}

	patchHelper, err := patch.NewHelper(infraCluster, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to init patch helper for infra Cluster %s", req.NamespacedName)
	}

	defer func() {
		patchOpts := []patch.Option{
			patch.WithOwnedConditions{Conditions: []capiv1.ConditionType{
				infrav1.CredentialsReadyCondition,
			}},
		}
		if reterr == nil {
			patchOpts = append(patchOpts, patch.WithStatusObservedGeneration{})
		}

		if reterr != nil {
			infraCluster.Status.Phase = string(infrav1.ClusterPhaseFailed)
		}

		if err := patchHelper.Patch(ctx, infraCluster, patchOpts...); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch infra Cluster %s", req.NamespacedName)})
		}
	}()

	// Add finalizer if not exist to void the race condition.
	if !controllerutil.ContainsFinalizer(infraCluster, clusterFinalizer) {
		infraCluster.Status.Phase = string(capiv1.ClusterPhasePending)
		controllerutil.AddFinalizer(infraCluster, clusterFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deletion reconciliation loop.
	if !infraCluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if infraCluster.Status.Phase != string(infrav1.ClusterPhaseDeleting) {
			infraCluster.Status.Phase = string(infrav1.ClusterPhaseDeleting)
			return ctrl.Result{}, nil
		}

		ctxLogger.Info("Reconciling deletion for cluster")
		return r.reconcileDelete(ctx, infraCluster)
	}

	// Handle normal loop.
	infraCluster.Status.Phase = string(infrav1.ClusterPhaseProvisioning)
	return r.reconcile(ctx, infraCluster)
}

func (r *ClusterController) reconcileDelete(ctx context.Context, infraCluster *infrav1.Cluster) (ctrl.Result, error) {
	if err := r.deleteCAPIClusterIfNeeded(ctx, infraCluster); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to delete CAPI Cluster %s/%s", infraCluster.Namespace, infraCluster.Name)
	}

	capiCluster := &capiv1.Cluster{}
	getClusterErr := r.Get(ctx, types.NamespacedName{Namespace: infraCluster.Namespace, Name: infraCluster.Name}, capiCluster)
	if !apiserrors.IsNotFound(getClusterErr) {
		// retry before CAPI Cluster is deleted
		return ctrl.Result{RequeueAfter: r.PollInterval}, nil
	}

	switch infraCluster.Spec.InfraType {
	case infrav1.AWSClusterInfraType:
		if err := r.reconcileDeleteAWS(ctx, infraCluster); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to delete AWS Cluster %s/%s", infraCluster.Namespace, infraCluster.Name)
		}
	default:
		// do nothing
	}

	// clean up ClusterResourceSet
	if err := r.deleteClusterResourceSets(ctx, infraCluster); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to delete ClusterResourceSet for Cluster %s/%s", infraCluster.Namespace, infraCluster.Name)
	}

	// Remove finalizer when all related resources are deleted.
	controllerutil.RemoveFinalizer(infraCluster, clusterFinalizer)
	return ctrl.Result{}, nil
}

func (r *ClusterController) deleteClusterResourceSets(ctx context.Context, infraCluster *infrav1.Cluster) error {
	csrList := &addonsv1.ClusterResourceSetList{}
	if err := r.List(ctx, csrList, client.InNamespace(infraCluster.Namespace), clusterMatchingLabels(infraCluster)); err != nil {
		return errors.Wrapf(err, "failed to list ClusterResourceSet")
	}

	for _, csr := range csrList.Items {
		if err := r.Delete(ctx, &csr); err != nil {
			if apiserrors.IsNotFound(err) {
				continue
			}

			return errors.Wrapf(err, "failed to delete ClusterResourceSet %s/%s", csr.Namespace, csr.Name)
		}
	}

	return nil
}

func (r *ClusterController) deleteCAPIClusterIfNeeded(ctx context.Context, infraCluster *infrav1.Cluster) error {
	cluster := &capiv1.Cluster{}
	nn := types.NamespacedName{
		Namespace: infraCluster.Namespace,
		Name:      infraCluster.Name,
	}
	if err := r.Get(ctx, nn, cluster); err != nil {
		if apiserrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "failed to get CAPI Cluster %s/%s", infraCluster.Namespace, infraCluster.Name)
	}

	if err := r.Delete(ctx, cluster); err != nil {
		if apiserrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "failed to delete CAPI Cluster %s/%s", infraCluster.Namespace, infraCluster.Name)
	}

	return nil
}

func (r *ClusterController) reconcile(ctx context.Context, infraCluster *infrav1.Cluster) (ctrl.Result, error) {
	switch infraCluster.Spec.InfraType {
	case infrav1.AWSClusterInfraType:
		if _, err := r.reconcileAWS(ctx, infraCluster); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile AWS Cluster %s/%s", infraCluster.Namespace, infraCluster.Name)
		}
	default:
		return ctrl.Result{}, errors.Errorf("unsupported infra type %s", infraCluster.Spec.InfraType)
	}

	// set ClusterResourceSet's owner reference to infraCluster
	if err := r.reconcileClusterResourceSet(ctx, infraCluster); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile ClusterResourceSet for infra Cluster %s/%s", infraCluster.Namespace, infraCluster.Name)
	}
	infraCluster.Status.Phase = string(infrav1.ClusterPhaseProvisioned)

	// check Cluster status
	if !r.clusterReady(ctx, infraCluster) {
		return ctrl.Result{RequeueAfter: r.PollInterval}, nil
	}

	infraCluster.Status.Phase = string(infrav1.ClusterPhaseReady)
	conditions.MarkTrue(infraCluster, capiv1.ReadyCondition)
	return ctrl.Result{}, nil
}

// TODO: make this more gerneic, support other control plane providers
func (r *ClusterController) clusterReady(ctx context.Context, infraCluster *infrav1.Cluster) bool {
	capiCluster := &capiv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: infraCluster.Namespace, Name: infraCluster.Name}, capiCluster); err != nil {
		return false
	}

	kcp := &controlplanev1.KubeadmControlPlane{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: capiCluster.Namespace, Name: capiCluster.Spec.ControlPlaneRef.Name}, kcp); err != nil {
		return false
	}

	if kcp.Status.Initialized && kcp.Status.Ready {
		return true
	}

	return false
}

func (r *ClusterController) reconcileClusterResourceSet(ctx context.Context, infraCluster *infrav1.Cluster) error {
	// reconcile OwnerReferences for ClusterResourceSet
	csrList := &addonsv1.ClusterResourceSetList{}
	if err := r.List(ctx, csrList, clusterMatchingLabels(infraCluster)); err != nil {
		return errors.Wrapf(err, "failed to list ClusterResourceSet for infra Cluster %s/%s", infraCluster.Namespace, infraCluster.Name)
	}

	ownerRef := metav1.OwnerReference{
		APIVersion: infrav1.GroupVersion.String(),
		Kind:       "Cluster",
		Name:       infraCluster.Name,
		UID:        infraCluster.UID,
	}
	for _, csr := range csrList.Items {
		if err := r.ensureClusterResourceSetRefsOwnerRef(ctx, &csr, ownerRef); err != nil {
			return errors.Wrapf(err, "failed to delete refs for ClusterResourceSet %s/%s", csr.Namespace, csr.Name)
		}

		// always get 429 if update owner reference of ClusterResourceSet, so manually delete when deleting cluster
	}

	return nil
}

func (r *ClusterController) ensureClusterResourceSetRefsOwnerRef(ctx context.Context, csr *addonsv1.ClusterResourceSet, ownerRef metav1.OwnerReference) error {
	for _, ref := range csr.Spec.Resources {
		if ref.Kind == "ConfigMap" {
			cm := &corev1.ConfigMap{}
			nn := types.NamespacedName{Namespace: csr.Namespace, Name: ref.Name}
			if err := r.Get(ctx, nn, cm); err != nil {
				if apiserrors.IsNotFound(err) {
					continue
				}
				return errors.Wrapf(err, "failed to get ConfigMap %s/%s", csr.Namespace, ref.Name)
			}

			cm.OwnerReferences = capiutil.EnsureOwnerRef(cm.OwnerReferences, ownerRef)
			if err := r.Update(ctx, cm); err != nil {
				return errors.Wrapf(err, "failed to update ConfigMap %s/%s", csr.Namespace, ref.Name)
			}
		}

		if ref.Kind == "Secret" {
			secret := &corev1.Secret{}
			nn := types.NamespacedName{Namespace: csr.Namespace, Name: ref.Name}
			if err := r.Get(ctx, nn, secret); err != nil {
				if apiserrors.IsNotFound(err) {
					continue
				}
				return errors.Wrapf(err, "failed to get Secret %s/%s", csr.Namespace, ref.Name)
			}

			secret.OwnerReferences = capiutil.EnsureOwnerRef(secret.OwnerReferences, ownerRef)
			if err := r.Update(ctx, secret); err != nil {
				return errors.Wrapf(err, "failed to update Secret %s/%s", csr.Namespace, ref.Name)
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.Cluster{}).
		Complete(r)
}
