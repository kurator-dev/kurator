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
	apiserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
)

const (
	// ClusterFinalizer allows ClusterController to clean up associated resources before removing it from apiserver.
	clusterFinalizer = "cluster.infra.kurator.dev"
)

var (
	// TODO: make this configurable
	pollInterval = 10 * time.Second
	pollTimeout  = 5 * time.Minute
)

// ClusterController reconciles a Cluster object
type ClusterController struct {
	client.Client
	Scheme *runtime.Scheme
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
		if err := patchHelper.Patch(ctx, infraCluster); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch infra Cluster %s", req.NamespacedName)})
		}
	}()

	// Add finalizer if not exist to void the race condition.
	if !controllerutil.ContainsFinalizer(infraCluster, clusterFinalizer) {
		controllerutil.AddFinalizer(infraCluster, clusterFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deletion reconciliation loop.
	if !infraCluster.ObjectMeta.DeletionTimestamp.IsZero() {
		ctxLogger.Info("Reconciling deletion for cluster")
		return r.reconcileDelete(ctx, infraCluster)
	}

	// Handle normal loop.
	return r.reconcile(ctx, infraCluster)
}

func (r *ClusterController) reconcileDelete(ctx context.Context, infraCluster *infrav1.Cluster) (ctrl.Result, error) {
	if err := r.deleteCAPIClusterIfNeeded(ctx, infraCluster); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to delete CAPI Cluster %s/%s", infraCluster.Namespace, infraCluster.Name)
	}

	nn := types.NamespacedName{Namespace: infraCluster.Namespace, Name: infraCluster.Name}
	capiCluster := &capiv1.Cluster{}
	if err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		getClusterErr := r.Get(ctx, nn, capiCluster)
		if getClusterErr != nil && apiserrors.IsNotFound(getClusterErr) {
			// return when capiv1.Cluster is deleted
			return true, nil
		}

		return false, nil
	}); err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to wait for CAPI Cluster %s/%s to be deleted", infraCluster.Namespace, infraCluster.Name)
	}

	// TODO: implement deletion logic of infra resources.

	// Remove finalizer when all related resources are deleted.
	controllerutil.RemoveFinalizer(infraCluster, clusterFinalizer)
	return ctrl.Result{}, nil
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
		// TODO: implement AWS reconcile logic
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, errors.Errorf("unsupported infra type %s", infraCluster.Spec.InfraType)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.Cluster{}).
		Complete(r)
}
