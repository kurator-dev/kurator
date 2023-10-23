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

	"github.com/robfig/cron/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"kurator.dev/kurator/pkg/apis/backups/v1alpha1"
)

var _ webhook.CustomValidator = &BackupWebhook{}

type BackupWebhook struct {
	Client client.Reader
}

func (wh *BackupWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.Backup{}).
		WithValidator(wh).
		Complete()
}

func (wh *BackupWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	in, ok := obj.(*v1alpha1.Backup)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Backup but got a %T", obj))
	}

	return wh.validate(in)
}

func (wh *BackupWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	in, ok := newObj.(*v1alpha1.Backup)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Backup but got a %T", newObj))
	}

	return wh.validate(in)
}

func (wh *BackupWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (wh *BackupWebhook) validate(in *v1alpha1.Backup) error {
	var allErrs field.ErrorList

	// Validate Schedule
	if len(in.Spec.Schedule) != 0 {
		if _, err := cron.ParseStandard(in.Spec.Schedule); err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("schedule"), in.Spec.Schedule, fmt.Sprintf("invalid cron expression: %s", err)))
		}
	}

	// Validate referenced clusters in destination
	if len(in.Spec.Destination.Clusters) != 0 {
		// Validate referenced clusters in destination
		allErrs = append(allErrs, validateDestinationClusters(in.Spec.Destination.Clusters)...)
	}

	// Validate Policy
	if in.Spec.Policy != nil {
		// Validate TTL
		if in.Spec.Policy.TTL.Duration < 0 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("policy", "ttl"), in.Spec.Policy.TTL, "TTL cannot be negative"))
		}
		// Validate Resource Filter
		allErrs = append(allErrs, validateResourceFilter(in.Spec.Policy.ResourceFilter)...)
	}

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha1.SchemeGroupVersion.WithKind("Backup").GroupKind(), in.Name, allErrs)
	}

	return nil
}
