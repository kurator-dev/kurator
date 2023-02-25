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
	"net"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/cluster-api/util/version"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	v1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
)

var _ webhook.CustomValidator = &ClusterWebhook{}

type ClusterWebhook struct {
	Client client.Reader
}

func (wh *ClusterWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1.Cluster{}).
		WithValidator(wh).
		Complete()
}

func (wh *ClusterWebhook) ValidateCreate(_ context.Context, obj runtime.Object) error {
	in, ok := obj.(*v1.Cluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Cluster but got a %T", obj))
	}

	return wh.validate(in)
}

func validateInfra(in *v1.Cluster) field.ErrorList {
	var allErrs field.ErrorList
	if in.Spec.InfraType == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "infraType"), "must be set"))
	}

	if in.Spec.InfraType == v1.AWSClusterInfraType && in.Spec.Region == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "region"), "must be set"))
	}

	if in.Spec.Credential == nil || in.Spec.Credential.SecretRef == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "credential", "secretRef"), "must be set"))
	}

	return allErrs
}

func validateVersion(in *v1.Cluster) field.ErrorList {
	var allErrs field.ErrorList
	if in.Spec.Version == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "version"), "must be set"))
	} else {
		if !version.KubeSemver.MatchString(in.Spec.Version) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "version"), in.Spec.Version, "must be a valid Kubernetes version"))
		}
	}

	return allErrs
}

func validateNetwork(network v1.NetworkConfig, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, validateCIDR(network.VPC.CIDRBlock, fldPath.Child("vpc", "cidrBlock"))...)
	allErrs = append(allErrs, validateCIDRBlocks(network.PodCIDRs, fldPath.Child("podCIDRs"))...)
	allErrs = append(allErrs, validateCIDRBlocks(network.ServiceCIDRs, fldPath.Child("serviceCIDRs"))...)

	return allErrs
}

func validateCIDR(cidr string, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if _, _, err := net.ParseCIDR(cidr); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath, cidr, err.Error()))
	}
	return allErrs
}

func validateCIDRBlocks(cidrBlocks []string, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	for i, cidr := range cidrBlocks {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Index(i), cidr, err.Error()))
		}
	}

	return allErrs
}

const (
	AWSMaxDataVolumeCount = 25
)

func validateVolume(in *v1.Cluster) field.ErrorList {
	var allErrs field.ErrorList
	if in.Spec.InfraType == v1.AWSClusterInfraType {
		allErrs = append(allErrs, validateAWSVolume(in)...)
	}

	return allErrs
}

func validateAWSVolume(in *v1.Cluster) field.ErrorList {
	var allErrs field.ErrorList

	if len(in.Spec.Master.NonRootVolumes) > AWSMaxDataVolumeCount {
		allErrs = append(allErrs, field.TooMany(
			field.NewPath("spec", "master", "nonRootVolumes"),
			len(in.Spec.Master.NonRootVolumes),
			AWSMaxDataVolumeCount))
	}

	workersFiledPath := field.NewPath("spec", "workers")
	for idx, node := range in.Spec.Workers {
		if len(node.NonRootVolumes) > AWSMaxDataVolumeCount {
			allErrs = append(allErrs, field.TooMany(
				workersFiledPath.Index(idx).Child("nonRootVolumes"),
				len(node.NonRootVolumes),
				AWSMaxDataVolumeCount))
		}
	}
	return allErrs
}

func validateAdditionalResource(in []v1.ResourceRef, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if len(in) == 0 {
		return allErrs
	}

	keys := sets.NewString()
	for i, resource := range in {
		if resource.Name == "" {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("name"), "must be set"))
		}
		if resource.Kind != "Secret" && resource.Kind != "ConfigMap" {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("type"), "must be either Secret or ConfigMap"))
		}
		key := fmt.Sprintf("%s/%s", resource.Kind, resource.Name)
		if keys.Has(key) {
			allErrs = append(allErrs, field.Duplicate(fldPath.Index(i), key))
		} else {
			keys.Insert(key)
		}
	}

	return allErrs
}

func (wh *ClusterWebhook) validate(in *v1.Cluster) error {
	var allErrs field.ErrorList

	// The Cluste name is used as a label value, so it must be a valid label value.
	if errs := validation.IsValidLabelValue(in.Name); len(errs) > 0 {
		for _, err := range errs {
			allErrs = append(allErrs, field.Invalid(field.NewPath("metadata", "name"), in.Name, fmt.Sprintf("must be a valid label value: %s", err)))
		}
	}

	allErrs = append(allErrs, validateInfra(in)...)
	allErrs = append(allErrs, validateVersion(in)...)
	allErrs = append(allErrs, validateNetwork(in.Spec.Network, field.NewPath("spec", "network"))...)
	allErrs = append(allErrs, validateVolume(in)...)
	allErrs = append(allErrs, validateAdditionalResource(in.Spec.AdditionalResources, field.NewPath("spec", "additionalResources"))...)

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(v1.SchemeGroupVersion.WithKind("Cluster").GroupKind(), in.Name, allErrs)
	}
	return nil
}

func (wh *ClusterWebhook) ValidateUpdate(_ context.Context, obj runtime.Object, _ runtime.Object) error {
	in, ok := obj.(*v1.Cluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Cluster but got a %T", obj))
	}

	return wh.validate(in)
}

func (wh *ClusterWebhook) ValidateDelete(_ context.Context, obj runtime.Object) error {
	return nil
}
