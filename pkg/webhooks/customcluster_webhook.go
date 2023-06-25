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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"kurator.dev/kurator/pkg/apis/infra/v1alpha1"
)

var _ webhook.CustomValidator = &CustomClusterWebhook{}

type CustomClusterWebhook struct {
	Client client.Reader
}

func (wh *CustomClusterWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.CustomCluster{}).
		WithValidator(wh).
		Complete()
}

func (wh *CustomClusterWebhook) ValidateCreate(_ context.Context, obj runtime.Object) error {
	in, ok := obj.(*v1alpha1.CustomCluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a CustomCluster but got a %T", obj))
	}

	return wh.validate(in)
}

func (wh *CustomClusterWebhook) validate(in *v1alpha1.CustomCluster) error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validateCNI(in)...)
	allErrs = append(allErrs, validateMachineRef(in.Spec.MachineRef)...)
	if in.Spec.ControlPlaneConfig != nil {
		allErrs = append(allErrs, validateControlPlaneConfig(in.Spec.ControlPlaneConfig)...)
	}

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha1.SchemeGroupVersion.WithKind("CustomCluster").GroupKind(), in.Name, allErrs)
	}

	return nil
}

var validCNIs = []string{"calico", "canal", "cilium", "flannel", "kube-ovn", "kube-router", "macvlan", "weave"}

func IsValidCNI(value string) bool {
	for _, v := range validCNIs {
		if value == v {
			return true
		}
	}
	return false
}

func validateCNI(in *v1alpha1.CustomCluster) field.ErrorList {
	var allErrs field.ErrorList
	if !IsValidCNI(in.Spec.CNI.Type) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "cni", "type"), in.Spec.CNI.Type,
			fmt.Sprintf("invalid CNI type: %v ,it should be one of %v ", in.Spec.CNI.Type, validCNIs)))
	}

	return allErrs
}

func validateMachineRef(machineRef *corev1.ObjectReference) field.ErrorList {
	var allErrs field.ErrorList

	allErrs = append(allErrs, ValidateObjectReference(machineRef, field.NewPath("spec", "machineRef"))...)

	return allErrs
}

func validateControlPlaneConfig(in *v1alpha1.ControlPlaneConfig) field.ErrorList {
	var allErrs field.ErrorList

	if len(in.Address) != 0 {
		allErrs = append(allErrs, validateIP(in.Address, field.NewPath("spec", "controlPlaneConfig", "address"))...)
	}

	return allErrs
}

// ValidateUpdate is not checking for changes in parameters such as cni.type, api address, certSANs, and so on.
// These parameters are set during cluster initialization and are not expected to change during the lifecycle of the cluster.
// Altering these values does not impact the system because these parameters are not re-checked after cluster creation.
func (wh *CustomClusterWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	_, ok := oldObj.(*v1alpha1.CustomCluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a CustomCluster but got a %T", oldObj))
	}

	newCustomCluster, ok := newObj.(*v1alpha1.CustomCluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a CustomCluster but got a %T", newObj))
	}

	return wh.validate(newCustomCluster)
}

func (wh *CustomClusterWebhook) ValidateDelete(_ context.Context, obj runtime.Object) error {
	return nil
}
