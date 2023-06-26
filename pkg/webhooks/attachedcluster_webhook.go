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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
)

var _ webhook.CustomValidator = &AttachedClusterWebhook{}

type AttachedClusterWebhook struct {
	Client client.Reader
}

func (wh *AttachedClusterWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.AttachedCluster{}).
		WithValidator(wh).
		Complete()
}

func (wh *AttachedClusterWebhook) ValidateCreate(_ context.Context, obj runtime.Object) error {
	in, ok := obj.(*v1alpha1.AttachedCluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a AttachedCluster but got a %T", obj))
	}

	return wh.validate(in)
}

func (wh *AttachedClusterWebhook) validate(in *v1alpha1.AttachedCluster) error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validateSecretKeyRef(in.Spec.Kubeconfig)...)

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha1.SchemeGroupVersion.WithKind("AttachedCluster").GroupKind(), in.Name, allErrs)
	}

	return nil
}

func validateSecretKeyRef(kubeconfig v1alpha1.SecretKeyRef) field.ErrorList {
	var allErrs field.ErrorList

	if kubeconfig.Name == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "kubeconfig", "name"), "must be set"))
	} else {
		allErrs = append(allErrs, validateDNS1123Domain(kubeconfig.Name, field.NewPath("spec", "kubeconfig", "name"))...)
	}
	if kubeconfig.Key == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "kubeconfig", "key"), "must be set"))
	} else {
		// The IsDNS1123Subdomain is used to validate the keys due to the portability and utility of this format across various contexts.
		allErrs = append(allErrs, validateDNS1123Domain(kubeconfig.Key, field.NewPath("spec", "kubeconfig", "key"))...)
	}

	return allErrs
}

func (wh *AttachedClusterWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	_, ok := oldObj.(*v1alpha1.AttachedCluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a AttachedCluster but got a %T", oldObj))
	}

	newAttachedCluster, ok := newObj.(*v1alpha1.AttachedCluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a AttachedCluster but got a %T", newObj))
	}

	return wh.validate(newAttachedCluster)
}

func (wh *AttachedClusterWebhook) ValidateDelete(_ context.Context, obj runtime.Object) error {
	return nil
}
