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

package fleet

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	clusterv1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

const (
	FleetKind      = "Fleet"
	FleetFinalizer = "fleet.kurator.dev"

	PhaseRunning            = "Running"
	PhaseFailed             = "Failed"
	PhaseReady              = "Ready"
	PhaseTerminating        = "Terminating"
	PhaseTerminateSucceeded = "TerminateSucceeded"
	PhaseTerminateFailed    = "TerminateFailed"
)

const RequeueAfter = 5 * time.Second
const FleetLabel = "fleet.kurator.dev/fleet-name"

// FleetManager reconciles a Cluster object
type FleetManager struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (f *FleetManager) SetupWithManager(mgr ctrl.Manager) error {
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&fleetapi.Fleet{}).
		Build(f)
	if err != nil {
		return err
	}

	if err := c.Watch(
		&source.Kind{Type: &corev1.Pod{}},
		handler.EnqueueRequestsFromMapFunc(f.objectToFleetFunc),
	); err != nil {
		return fmt.Errorf("failed adding Watch for Secret to controller manager: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &clusterv1alpha1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(f.objectToFleetFunc),
	); err != nil {
		return fmt.Errorf("failed adding Watch for Secret to controller manager: %v", err)
	}

	return nil
}

func (f *FleetManager) objectToFleetFunc(o client.Object) []ctrl.Request {
	labels := o.GetLabels()
	if labels[FleetLabel] != "" {
		return []ctrl.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: o.GetNamespace(),
					Name:      labels[FleetLabel],
				},
			},
		}
	}

	return nil
}

func (f *FleetManager) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	fleet := &fleetapi.Fleet{}
	if err := f.Get(ctx, req.NamespacedName, fleet); err != nil {
		if apiserrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrapf(err, "failed to get fleet %s", req.NamespacedName)
	}

	patchHelper, err := patch.NewHelper(fleet, f.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to init patch helper for fleet %s", req.NamespacedName)
	}

	defer func() {
		patchOpts := []patch.Option{}
		if err := patchHelper.Patch(ctx, fleet, patchOpts...); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch fleet %s", req.NamespacedName)})
		}
	}()

	// Add finalizer if not exist to void the race condition.
	if !controllerutil.ContainsFinalizer(fleet, FleetFinalizer) {
		fleet.Status.Phase = PhaseRunning
		controllerutil.AddFinalizer(fleet, FleetFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deletion reconciliation loop.
	if fleet.DeletionTimestamp != nil {
		if fleet.Status.Phase != PhaseTerminating {
			fleet.Status.Phase = PhaseTerminating
		}

		return f.reconcileDelete(ctx, fleet)
	}

	// Handle normal loop.
	return f.reconcile(ctx, fleet)
}

func (f *FleetManager) reconcile(ctx context.Context, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log = log.WithValues("fleet", types.NamespacedName{Name: fleet.Name, Namespace: fleet.Namespace})

	// Install fleet control plane
	if err := f.reconcileControlPlane(ctx, fleet); err != nil {
		log.Error(err, "controlplane reconcile failed")
		fleet.Status.Phase = PhaseFailed
		fleet.Status.Reason = err.Error()
		return ctrl.Result{}, err
	}

	if fleet.Status.Phase != PhaseReady {
		return ctrl.Result{}, nil
	}

	// Loop over all clusters and reconcile them.
	res, err := f.reconcileClusters(ctx, fleet)
	if err != nil || res.RequeueAfter > 0 {
		return res, err
	}

	return ctrl.Result{}, nil
}

func (f *FleetManager) reconcileDelete(ctx context.Context, fleet *fleetapi.Fleet) (ctrl.Result, error) {
	if res, err := f.reconcileClustersOnDelete(ctx, fleet); err != nil {
		return res, err
	}

	// Delete fleet control plane
	if err := f.deleteControlPlane(ctx, fleet); err != nil {
		return ctrl.Result{}, err
	}

	if fleet.Status.Phase == PhaseTerminateSucceeded {
		// Remove finalizer when all related resources are deleted.
		controllerutil.RemoveFinalizer(fleet, FleetFinalizer)
	}
	return ctrl.Result{}, nil
}
