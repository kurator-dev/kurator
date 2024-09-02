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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"kurator.dev/kurator/pkg/apis/apps/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

var _ webhook.CustomValidator = &ApplicationWebhook{}
var _ webhook.CustomDefaulter = &ApplicationWebhook{}

type ApplicationWebhook struct {
	Client client.Reader
}

func (wh *ApplicationWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.Application{}).
		WithValidator(wh).
		WithDefaulter(wh).
		Complete()
}

func (wh *ApplicationWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	in, ok := obj.(*v1alpha1.Application)
	log := ctrl.LoggerFrom(ctx)
	log.Info("All field Validate succeed")
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a Application but got a %T", obj))
	}

	return nil, wh.validate(in)
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
			isNil            = false
		)
		for i, policy := range in.Spec.SyncPolicies {
			// if individual policy fleet is not set, return err
			if policy.Destination == nil || policy.Destination.Fleet == "" {
				isNil = true
				continue
			}
			if isFirst {
				firstPolicyFleet = policy.Destination.Fleet
				isFirst = false
			}
			if !isFirst && firstPolicyFleet != policy.Destination.Fleet {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "syncPolicies").Index(i).Child("destination", "fleet"), policy.Destination.Fleet, fmt.Sprintf("must be same as firstPolicyFleet:%v, because fleet must be consistent throughout the application", firstPolicyFleet)))
			}
		}
		if isNil && !isFirst {
			allErrs = append(allErrs, field.Required(field.NewPath("spec", "syncPolicies").Child("destination", "fleet"), "must be set when application.spec.destination.fleet is not set"))
		}
	}

	return allErrs
}

func (wh *ApplicationWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	_, ok := oldObj.(*v1alpha1.Application)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a Application but got a %T", oldObj))
	}

	newApplication, ok := newObj.(*v1alpha1.Application)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a Application but got a %T", newObj))
	}

	return nil, wh.validate(newApplication)
}

func (wh *ApplicationWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (wh *ApplicationWebhook) Default(ctx context.Context, obj runtime.Object) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("setting default")
	app, ok := obj.(*v1alpha1.Application)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Application but got a %T", obj))
	}
	if err := defaultApp(ctx, app); err != nil {
		return err
	}
	return nil
}

func defaultApp(ctx context.Context, app *v1alpha1.Application) error {
	log := ctrl.LoggerFrom(ctx)
	log = log.WithValues("application", types.NamespacedName{Name: app.Name, Namespace: app.Namespace})
	if err := defaultSyncPolicies(ctx, app.Spec.SyncPolicies); err != nil {
		return err
	}
	log.Info("All field set default")
	return nil
}

func defaultSyncPolicies(ctx context.Context, SyncPolicies []*v1alpha1.ApplicationSyncPolicy) error {
	log := ctrl.LoggerFrom(ctx)
	log = log.WithValues("application")
	for i, syncPoliciy := range SyncPolicies {
		if syncPoliciy.Rollout == nil {
			continue
		}
		switch syncPoliciy.Rollout.TrafficRoutingProvider {
		case fleetapi.Nginx:
			if syncPoliciy.Rollout.RolloutPolicy.TrafficRouting.Host == "" {
				return apierrors.NewBadRequest(fmt.Sprintf("expected application.syncPolicies.Rollout.RolloutPolicy.TrafficRouting.Host in index %d", i))
			}
		case fleetapi.Kuma:
			if syncPoliciy.Rollout.RolloutPolicy.TrafficRouting.Protocol == "" {
				syncPoliciy.Rollout.RolloutPolicy.TrafficRouting.Protocol = "http"
			}
		}
	}
	log.Info("set SyncPolicies default success")
	return nil
}
