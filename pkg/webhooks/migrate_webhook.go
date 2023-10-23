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

	"kurator.dev/kurator/pkg/apis/backups/v1alpha1"
)

var _ webhook.CustomValidator = &MigrateWebhook{}

type MigrateWebhook struct {
	Client client.Reader
}

func (wh *MigrateWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.Migrate{}).
		WithValidator(wh).
		Complete()
}

func (wh *MigrateWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	in, ok := obj.(*v1alpha1.Migrate)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Migrate but got a %T", obj))
	}

	return wh.validate(in)
}

func (wh *MigrateWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	in, ok := newObj.(*v1alpha1.Migrate)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Migrate but got a %T", newObj))
	}

	return wh.validate(in)
}

func (wh *MigrateWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

func (wh *MigrateWebhook) validate(in *v1alpha1.Migrate) error {
	var allErrs field.ErrorList

	// Ensure that SourceCluster points to only ONE cluster.
	// Because the current migration only supports migrating from one SourceCluster to one or more TargetCluster.
	if len(in.Spec.SourceCluster.Clusters) != 1 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("sourceCluster"), len(in.Spec.SourceCluster.Clusters), "must have exactly one source cluster"))
	}
	// Validate referenced clusters in SourceCluster
	allErrs = append(allErrs, validateDestinationClusters(in.Spec.SourceCluster.Clusters)...)

	sourceCluster := in.Spec.SourceCluster.Clusters[0]

	// If the 'clusters' field is not specified, it defaults to encompassing all clusters; hence, there's no need to validate the count of TargetClusters.

	// Validate referenced clusters in TargetClusters
	allErrs = append(allErrs, validateDestinationClusters(in.Spec.TargetClusters.Clusters)...)

	// Ensure target cluster not be the same as source cluster
	for _, targetCluster := range in.Spec.TargetClusters.Clusters {
		if targetCluster.Name == sourceCluster.Name && targetCluster.Kind == sourceCluster.Kind {
			allErrs = append(allErrs, field.Invalid(field.NewPath("targetCluster"), targetCluster.Name, "target cluster cannot be the same as source cluster"))
		}
	}

	// Validate Policy
	if in.Spec.Policy != nil {
		// Validate Resource Filter
		allErrs = append(allErrs, validateResourceFilter(in.Spec.Policy.ResourceFilter)...)
	}

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(v1alpha1.SchemeGroupVersion.WithKind("Migrate").GroupKind(), in.Name, allErrs)
	}

	return nil
}
