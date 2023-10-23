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
	"fmt"
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"kurator.dev/kurator/pkg/apis/backups/v1alpha1"
)

func validateIP(IP string, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if net.ParseIP(IP) == nil {
		allErrs = append(allErrs, field.Invalid(fldPath, IP, fmt.Sprintf("invalid ip address: %s", IP)))
	}
	return allErrs
}

// validateDNS1123Label checks if a string is a valid DNS1123 label.
// A lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-',
// must start and end with an alphanumeric character,
// and must have a maximum length of 63 characters.
func validateDNS1123Label(label string, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if errs := validation.IsDNS1123Label(label); len(errs) > 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, label, strings.Join(errs, "; ")))
	}
	return allErrs
}

// validateDNS1123Domain checks if a string is a valid DNS1123 domain.
// A lowercase RFC 1123 domain must consist of lower case alphanumeric characters, '-' or '.'
// must start and end with an alphanumeric character,
// and must have a maximum length of 253 characters.
func validateDNS1123Domain(domain string, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if errs := validation.IsDNS1123Subdomain(domain); len(errs) > 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, domain, strings.Join(errs, "; ")))
	}
	return allErrs
}

func validateDestinationClusters(clusters []*corev1.ObjectReference) field.ErrorList {
	var allErrs field.ErrorList

	for _, clusterRef := range clusters {
		if len(clusterRef.Name) == 0 {
			allErrs = append(allErrs, field.Required(field.NewPath("destination", "clusters", "name"), "Name of the referenced cluster is required"))
		}
		if clusterRef.Kind == "" {
			allErrs = append(allErrs, field.Required(field.NewPath("destination", "clusters", "kind"), "Kind of the referenced cluster is required"))
		}
	}

	return allErrs
}

// validateResourceFilter remains the same as before, just moved to this file
func validateResourceFilter(filter *v1alpha1.ResourceFilter) field.ErrorList {
	var allErrs field.ErrorList

	if filter == nil {
		return allErrs
	}

	// Validate IncludedNamespaces and ExcludedNamespaces
	for _, includedNS := range filter.IncludedNamespaces {
		for _, excludedNS := range filter.ExcludedNamespaces {
			if includedNS == excludedNS {
				allErrs = append(allErrs, field.Invalid(field.NewPath("resourceFilter", "includedNamespaces"), includedNS, "Namespace is already in excludedNamespaces"))
			}
		}
	}

	// Validate IncludedResources and ExcludedResources
	for _, includedRes := range filter.IncludedResources {
		for _, excludedRes := range filter.ExcludedResources {
			if includedRes == excludedRes {
				allErrs = append(allErrs, field.Invalid(field.NewPath("resourceFilter", "includedResources"), includedRes, "Resource is already in excludedResources"))
			}
		}
	}

	// Validate IncludedClusterScopedResources and ExcludedClusterScopedResources
	for _, includedClusterRes := range filter.IncludedClusterScopedResources {
		for _, excludedClusterRes := range filter.ExcludedClusterScopedResources {
			if includedClusterRes == excludedClusterRes {
				allErrs = append(allErrs, field.Invalid(field.NewPath("resourceFilter", "includedClusterScopedResources"), includedClusterRes, "ClusterScopedResource is already in excludedClusterScopedResources"))
			}
		}
	}

	// Validate IncludedNamespaceScopedResources and ExcludedNamespaceScopedResources
	for _, includedNamespaceRes := range filter.IncludedNamespaceScopedResources {
		for _, excludedNamespaceRes := range filter.ExcludedNamespaceScopedResources {
			if includedNamespaceRes == excludedNamespaceRes {
				allErrs = append(allErrs, field.Invalid(field.NewPath("resourceFilter", "includedNamespaceScopedResources"), includedNamespaceRes, "NamespaceScopedResource is already in excludedNamespaceScopedResources"))
			}
		}
	}

	// Check IncludeClusterResources against IncludedClusterScopedResources and ExcludedClusterScopedResources
	if filter.IncludeClusterResources != nil && *filter.IncludeClusterResources {
		if len(filter.IncludedClusterScopedResources) > 0 || len(filter.ExcludedClusterScopedResources) > 0 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("resourceFilter", "includeClusterResources"), *filter.IncludeClusterResources, "Cannot be set when IncludedClusterScopedResources or ExcludedClusterScopedResources is set"))
		}
	}

	return allErrs
}
