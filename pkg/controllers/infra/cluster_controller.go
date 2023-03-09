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
	"reflect"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	clusterv1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	infraprovider "kurator.dev/kurator/pkg/infra"
	infraplugin "kurator.dev/kurator/pkg/infra/plugin"
	"kurator.dev/kurator/pkg/infra/scope"
	"kurator.dev/kurator/pkg/infra/util"
)

const (
	// ClusterFinalizer allows ClusterController to clean up associated resources before removing it from apiserver.
	clusterFinalizer = "cluster.cluster.kurator.dev"
)

// ClusterController reconciles a Cluster object
type ClusterController struct {
	client.Client
	Scheme       *runtime.Scheme
	RequeueAfter time.Duration
}

func (r *ClusterController) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctxLogger := log.FromContext(ctx)

	cluster := &clusterv1alpha1.Cluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if apiserrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrapf(err, "failed to get cluster %s", req.NamespacedName)
	}

	patchHelper, err := patch.NewHelper(cluster, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to init patch helper for cluster %s", req.NamespacedName)
	}

	defer func() {
		patchOpts := []patch.Option{
			patch.WithOwnedConditions{Conditions: []capiv1.ConditionType{
				clusterv1alpha1.InfrastructureReadyCondition,
				clusterv1alpha1.CNICondition,
				clusterv1alpha1.ReadyCondition,
			}},
		}

		if reterr != nil {
			cluster.Status.Phase = string(clusterv1alpha1.ClusterPhaseFailed)
		}

		if err := patchHelper.Patch(ctx, cluster, patchOpts...); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch cluster %s", req.NamespacedName)})
		}
	}()

	// Add finalizer if not exist to void the race condition.
	if !controllerutil.ContainsFinalizer(cluster, clusterFinalizer) {
		cluster.Status.Phase = string(capiv1.ClusterPhaseProvisioning)
		controllerutil.AddFinalizer(cluster, clusterFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deletion reconciliation loop.
	if !cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if cluster.Status.Phase != string(clusterv1alpha1.ClusterPhaseDeleting) {
			cluster.Status.Phase = string(clusterv1alpha1.ClusterPhaseDeleting)
			return ctrl.Result{}, nil
		}

		ctxLogger.Info("Reconciling deletion for cluster")
		return r.reconcileDelete(ctx, cluster)
	}

	// Handle normal loop.
	return r.reconcile(ctx, cluster)
}

func (r *ClusterController) reconcileDelete(ctx context.Context, cluster *clusterv1alpha1.Cluster) (ctrl.Result, error) {
	if err := r.deleteCAPIClusterIfNeeded(ctx, cluster); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to delete CAPI Cluster %s/%s", cluster.Namespace, cluster.Name)
	}

	capiCluster := &capiv1.Cluster{}
	err := r.Get(ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}, capiCluster)
	if err == nil || !apiserrors.IsNotFound(err) {
		// retry before CAPI Cluster is deleted
		return ctrl.Result{RequeueAfter: r.RequeueAfter}, nil
	}
	// CAPI Cluster is deleted, do the rest

	scope := scope.NewCluster(cluster)
	prov := infraprovider.NewProvider(r.Client, scope)
	if err := prov.Clean(ctx); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to delete AWS Cluster %s/%s", cluster.Namespace, cluster.Name)
	}

	// clean up ClusterResourceSet
	if err := r.deleteClusterResourceSets(ctx, scope); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to delete ClusterResourceSet for Cluster %s/%s", cluster.Namespace, cluster.Name)
	}

	// Remove finalizer when all related resources are deleted.
	controllerutil.RemoveFinalizer(cluster, clusterFinalizer)
	return ctrl.Result{}, nil
}

func (r *ClusterController) deleteClusterResourceSets(ctx context.Context, scope *scope.Cluster) error {
	csrList := &addonsv1.ClusterResourceSetList{}
	if err := r.List(ctx, csrList, client.InNamespace(scope.Namespace), scope.MatchingLabels()); err != nil {
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

func (r *ClusterController) deleteCAPIClusterIfNeeded(ctx context.Context, infraCluster *clusterv1alpha1.Cluster) error {
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

func (r *ClusterController) reconcile(ctx context.Context, cluster *clusterv1alpha1.Cluster) (ctrl.Result, error) {
	// TODO: precheck
	scope := scope.NewCluster(cluster)
	provider := infraprovider.NewProvider(r.Client, scope)
	if err := provider.Reconcile(ctx); err != nil {
		conditions.MarkFalse(cluster, clusterv1alpha1.InfrastructureReadyCondition, clusterv1alpha1.InfrastructureProvisionFailedReason,
			capiv1.ConditionSeverityError, err.Error())
		return ctrl.Result{RequeueAfter: r.RequeueAfter}, errors.Wrapf(err, "failed to reconcile AWS Cluster %s/%s", cluster.Namespace, cluster.Name)
	}
	// check Cluster status
	if err := provider.IsInitialized(ctx); err != nil {
		conditions.MarkFalse(cluster, clusterv1alpha1.InfrastructureReadyCondition, clusterv1alpha1.InfrastructureNotReadyReason,
			capiv1.ConditionSeverityWarning, err.Error())
		return ctrl.Result{RequeueAfter: r.RequeueAfter}, nil
	}
	conditions.MarkTrue(cluster, clusterv1alpha1.InfrastructureReadyCondition)

	if err := r.reconcileCNI(scope); err != nil {
		conditions.MarkFalse(cluster, clusterv1alpha1.CNICondition, clusterv1alpha1.CNIProvisionFailedReason,
			capiv1.ConditionSeverityError, err.Error())
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile CNI resources")
	}

	if err := provider.IsReady(ctx); err != nil {
		conditions.MarkFalse(cluster, clusterv1alpha1.CNICondition, clusterv1alpha1.CNINotReadyReason,
			capiv1.ConditionSeverityWarning, err.Error())
		return ctrl.Result{RequeueAfter: r.RequeueAfter}, nil
	}
	conditions.MarkTrue(cluster, clusterv1alpha1.CNICondition)

	if err := r.reconcileAdditionalResources(ctx, cluster); err != nil {
		conditions.MarkFalse(cluster, clusterv1alpha1.ReadyCondition, clusterv1alpha1.ClusterResourceSetProvisionFailedReason,
			capiv1.ConditionSeverityError, err.Error())
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile additional resources for cluster %s/%s", cluster.Namespace, cluster.Name)
	}

	if err := r.ensureOwnerReference(ctx, scope, cluster); err != nil {
		conditions.MarkFalse(cluster, clusterv1alpha1.ReadyCondition, clusterv1alpha1.ClusterResourceSetProvisionFailedReason,
			capiv1.ConditionSeverityError, err.Error())
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile ClusterResourceSet for cluster %s/%s", cluster.Namespace, cluster.Name)
	}

	conditions.MarkTrue(cluster, capiv1.ReadyCondition)
	cluster.Status.Phase = string(clusterv1alpha1.ClusterPhaseReady)
	return ctrl.Result{}, nil
}

func (r *ClusterController) reconcileAdditionalResources(ctx context.Context, infraCluster *clusterv1alpha1.Cluster) error {
	if len(infraCluster.Spec.AdditionalResources) == 0 {
		return nil
	}
	refs := util.AdditionalResources(infraCluster)

	csr := &addonsv1.ClusterResourceSet{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: infraCluster.Namespace, Name: infraCluster.Name}, csr); err != nil {
		if apiserrors.IsNotFound(err) {
			csr = &addonsv1.ClusterResourceSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: infraCluster.Namespace,
					Name:      infraCluster.Name,
					Labels: map[string]string{
						scope.ClusterNameLabel:      infraCluster.Name,
						scope.ClusterNamespaceLabel: infraCluster.Namespace,
					},
				},
				Spec: addonsv1.ClusterResourceSetSpec{
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							scope.ClusterNameLabel:      infraCluster.Name,
							scope.ClusterNamespaceLabel: infraCluster.Namespace,
						},
					},
					Resources: refs,
				},
			}
			if err := r.Create(ctx, csr); err != nil {
				return errors.Wrapf(err, "failed to create ClusterResourceSet %s/%s", csr.Namespace, csr.Name)
			}
		} else {
			return errors.Wrapf(err, "failed to get ClusterResourceSet %s/%s", csr.Namespace, csr.Name)
		}
	}

	if reflect.DeepEqual(csr.Spec.Resources, refs) {
		return nil
	}

	csr.Spec.Resources = refs
	if err := r.Update(ctx, csr); err != nil {
		return errors.Wrapf(err, "failed to update ClusterResourceSet %s/%s", csr.Namespace, csr.Name)
	}

	return nil
}

func (r *ClusterController) ensureOwnerReference(ctx context.Context, scope *scope.Cluster, infraCluster *clusterv1alpha1.Cluster) error {
	// reconcile OwnerReferences for ClusterResourceSet
	csrList := &addonsv1.ClusterResourceSetList{}
	if err := r.List(ctx, csrList, scope.MatchingLabels()); err != nil {
		return errors.Wrapf(err, "failed to list ClusterResourceSet for cluster %s/%s", scope.Namespace, scope.Name)
	}

	ownerRef := metav1.OwnerReference{
		APIVersion: clusterv1alpha1.GroupVersion.String(),
		Kind:       "Cluster",
		Name:       infraCluster.Name,
		UID:        infraCluster.UID,
	}
	for _, csr := range csrList.Items {
		if err := r.ensureClusterResourceSetRefsOwnerRef(ctx, &csr, ownerRef); err != nil {
			return errors.Wrapf(err, "failed to delete refs for ClusterResourceSet %s/%s", csr.Namespace, csr.Name)
		}

		// always get 409 if update owner reference of ClusterResourceSet, so manually delete when deleting cluster
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

func (r *ClusterController) reconcileCNI(scopeCluster *scope.Cluster) error {
	// For now, use CusterResourceSet to apply the CNI resources
	cni, err := infraplugin.RenderCNI(scopeCluster)
	if err != nil {
		return errors.Wrapf(err, "failed to render CNI resources")
	}

	_, err = util.PatchResources(cni)
	if err != nil {
		return errors.Wrapf(err, "failed to apply CNI resources")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.Cluster{}).
		Complete(r)
}
