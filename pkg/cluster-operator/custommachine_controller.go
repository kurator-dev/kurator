/*
Copyright 2022-2025 Kurator Authors.

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
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/controllers/external"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"kurator.dev/kurator/pkg/apis/infra/v1alpha1"
)

// CustomMachineController reconciles a CustomMachine object
type CustomMachineController struct {
	client.Client
	APIReader       client.Reader
	Scheme          *runtime.Scheme
	externalTracker external.ObjectTracker
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomMachineController) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.CustomMachine{}).
		WithOptions(options).
		Complete(r)
}

func (r *CustomMachineController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("customMachine", req.NamespacedName)

	// Fetch the CustomMachine instance.
	customMachine := &v1alpha1.CustomMachine{}
	if err := r.Client.Get(ctx, req.NamespacedName, customMachine); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("customMachine is not exist")
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		return ctrl.Result{Requeue: true}, err
	}
	return r.reconcile(ctx, customMachine)
}

func (r *CustomMachineController) reconcile(ctx context.Context, customMachine *v1alpha1.CustomMachine) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	keyRef := customMachine.Spec.Master[0].SSHKey
	obj, err := external.Get(ctx, r.Client, keyRef, customMachine.Namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Could not find external object for CustomMachine, requeuing", "refGroupVersionKind", keyRef.GroupVersionKind(), "refName", keyRef.Name)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}
	// Initialize the patch helper.
	patchHelper, err := patch.NewHelper(customMachine, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Set external object ControllerReference to the Cluster.
	if err := controllerutil.SetControllerReference(customMachine, obj, r.Client.Scheme()); err != nil {
		return ctrl.Result{}, err
	}
	// Ensure we add a watcher to the external ssh key object.
	if err := r.externalTracker.Watch(log, obj, handler.EnqueueRequestForOwner(r.Client.Scheme(), r.Client.RESTMapper(), &v1alpha1.CustomMachine{})); err != nil {
		return ctrl.Result{}, err
	}
	machineReady := true
	customMachine.Status.Ready = &machineReady
	err = patchHelper.Patch(ctx, customMachine)
	return ctrl.Result{}, err
}
