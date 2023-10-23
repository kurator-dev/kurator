package webhooks

import (
	"context"
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"kurator.dev/kurator/pkg/apis/backups/v1alpha1"
	"kurator.dev/kurator/pkg/webhooks/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var _ webhook.CustomValidator = &RestoreWebhook{}

type RestoreWebhook struct {
	Client client.Reader
}

func (wh *RestoreWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.Restore{}).
		WithValidator(wh).
		Complete()
}

func (wh *RestoreWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	in, ok := obj.(*v1alpha1.Restore)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Restore but got a %T", obj))
	}

	return wh.validate(in)
}

func (wh *RestoreWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	in, ok := newObj.(*v1alpha1.Restore)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Restore but got a %T", newObj))
	}

	return wh.validate(in)
}

func (wh *RestoreWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (wh *RestoreWebhook) validate(in *v1alpha1.Restore) error {
	var allErrs field.ErrorList

	// Validate referenced clusters in destination
	allErrs = append(allErrs, validation.ValidateDestinationClusters(in.Spec.Destination.Clusters)...)

	// Validate Resource Filter
	allErrs = append(allErrs, validation.ValidateResourceFilter(in.Spec.Policy.ResourceFilter)...)

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha1.SchemeGroupVersion.WithKind("Restore").GroupKind(), in.Name, allErrs)
	}

	return nil
}
