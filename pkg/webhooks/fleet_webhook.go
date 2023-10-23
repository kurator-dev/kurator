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

package webhooks

import (
	"context"
	"fmt"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

var _ webhook.CustomValidator = &FleetWebhook{}

// Define a map to store mutex for each namespace
var nsLocks = make(map[string]*sync.Mutex)

// Valid cluster kinds
var validClusterKinds = []string{"Cluster", "AttachedCluster", "CustomCluster"}

type FleetWebhook struct {
	Client client.Reader
}

func (wh *FleetWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.Fleet{}).
		WithValidator(wh).
		Complete()
}

func (wh *FleetWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	in, ok := obj.(*v1alpha1.Fleet)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Fleet but got a %T", obj))
	}

	// Ensure only one Fleet instance in a namespace
	mutex := getOrCreateMutexForNamespace(in.Namespace)
	mutex.Lock()
	defer mutex.Unlock()

	// Check if Fleet instance already exists in the namespace
	existing := &v1alpha1.Fleet{}
	if err := wh.Client.Get(ctx, client.ObjectKey{Namespace: in.Namespace, Name: in.Name}, existing); err == nil {
		return apierrors.NewBadRequest(fmt.Sprintf("a Fleet instance already exists in namespace %s: %s", existing.Namespace, existing.Name))
	}

	return nil
}

// Utility function to get or create a mutex for a namespace
func getOrCreateMutexForNamespace(ns string) *sync.Mutex {
	if _, exists := nsLocks[ns]; !exists {
		nsLocks[ns] = &sync.Mutex{}
	}
	return nsLocks[ns]
}

func (wh *FleetWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	_, ok := oldObj.(*v1alpha1.Fleet)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Fleet but got a %T", oldObj))
	}

	newFleet, ok := newObj.(*v1alpha1.Fleet)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Fleet but got a %T", newObj))
	}

	return wh.validate(newFleet)
}

func (wh *FleetWebhook) ValidateDelete(_ context.Context, obj runtime.Object) error {
	return nil
}

func (wh *FleetWebhook) validate(in *v1alpha1.Fleet) error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validateCluster(in)...)

	allErrs = append(allErrs, validatePlugin(in)...)

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha1.SchemeGroupVersion.WithKind("Fleet").GroupKind(), in.Name, allErrs)
	}

	return nil
}

// validateCluster checks the Clusters field of FleetSpec
func validateCluster(in *v1alpha1.Fleet) field.ErrorList {
	var allErrs field.ErrorList
	clusterPath := field.NewPath("spec", "clusters")

	// Check each cluster reference
	for i, clusterRef := range in.Spec.Clusters {
		if clusterRef == nil {
			allErrs = append(allErrs, field.Required(clusterPath.Index(i), "cluster reference cannot be nil"))
			continue
		}
		if clusterRef.Name == "" {
			allErrs = append(allErrs, field.Required(clusterPath.Index(i).Child("name"), "name is required"))
		}
		if clusterRef.Kind == "" {
			allErrs = append(allErrs, field.Required(clusterPath.Index(i).Child("kind"), "kind is required"))
		} else if !isValidClusterKind(clusterRef.Kind) {
			allErrs = append(allErrs, field.Invalid(clusterPath.Index(i).Child("kind"), clusterRef.Kind, "unsupported cluster kind; please use AttachedCluster to manage your own cluster"))
		}
	}

	return allErrs
}

// isValidClusterKind checks if the given kind is a valid cluster kind
func isValidClusterKind(kind string) bool {
	for _, validKind := range validClusterKinds {
		if kind == validKind {
			return true
		}
	}
	return false
}

// validatePlugin checks the Plugin field of FleetSpec
func validatePlugin(in *v1alpha1.Fleet) field.ErrorList {
	var allErrs field.ErrorList

	// TODO: add plugin validation here

	return allErrs
}
