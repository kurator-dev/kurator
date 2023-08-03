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

	"kurator.dev/kurator/pkg/apis/apps/v1alpha1"
)

var _ webhook.CustomValidator = &ApplicationWebhook{}

type ApplicationWebhook struct {
	Client client.Reader
}

func (wh *ApplicationWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.Application{}).
		WithValidator(wh).
		Complete()
}

func (wh *ApplicationWebhook) ValidateCreate(_ context.Context, obj runtime.Object) error {
	in, ok := obj.(*v1alpha1.Application)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Application but got a %T", obj))
	}

	return wh.validate(in)
}

func (wh *ApplicationWebhook) validate(in *v1alpha1.Application) error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, validateFleet(in)...)

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha1.SchemeGroupVersion.WithKind("Application").GroupKind(), in.Name, allErrs)
	}

	return nil
}

// validateFleet validates the fleet in the application with the following rules:
// 1 if defaultFleet is set, make sure all policy fleet(if set) is same as the defaultFleet
// 2 if defaultFleet is not set, every individual policies must be set and must be same as the first policy fleet
func validateFleet(in *v1alpha1.Application) field.ErrorList {
	var allErrs field.ErrorList

	defaultFleet := ""
	if in.Spec.Destination != nil {
		defaultFleet = in.Spec.Destination.Fleet
	}

	// if defaultFleet is set, make sure all policy fleet(if set) is same as the defaultFleet
	if defaultFleet != "" {
		for i, policy := range in.Spec.SyncPolicies {
			if policy.Destination != nil && policy.Destination.Fleet != "" && defaultFleet != policy.Destination.Fleet {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "syncPolicies").Index(i).Child("destination", "fleet"), policy.Destination.Fleet, fmt.Sprintf("must be same as application.spec.destination.fleet:%v, because fleet must be consistent throughout the application", defaultFleet)))
			}
		}
	}

	// if defaultFleet is not set, every individual policies must be set and must be same as the first policy fleet
	if defaultFleet == "" {
		var (
			firstPolicyFleet string
			isFirst          = true
		)
		for i, policy := range in.Spec.SyncPolicies {
			// if individual policy fleet is not set, return err
			if policy.Destination == nil || policy.Destination.Fleet == "" {
				allErrs = append(allErrs, field.Required(field.NewPath("spec", "syncPolicies").Index(i).Child("destination", "fleet"), "must be set when application.spec.destination.fleet is not set"))
				return allErrs
			}
			if isFirst {
				firstPolicyFleet = policy.Destination.Fleet
				isFirst = false
			}
			if !isFirst && firstPolicyFleet != policy.Destination.Fleet {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "syncPolicies").Index(i).Child("destination", "fleet"), policy.Destination.Fleet, fmt.Sprintf("must be same as firstPolicyFleet:%v, because fleet must be consistent throughout the application", firstPolicyFleet)))
			}
		}
	}

	return allErrs
}

func (wh *ApplicationWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	_, ok := oldObj.(*v1alpha1.Application)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Application but got a %T", oldObj))
	}

	newApplication, ok := newObj.(*v1alpha1.Application)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Application but got a %T", newObj))
	}

	return wh.validate(newApplication)
}

func (wh *ApplicationWebhook) ValidateDelete(_ context.Context, obj runtime.Object) error {
	return nil
}
