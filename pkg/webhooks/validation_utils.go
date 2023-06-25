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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func validateIP(IP string, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if net.ParseIP(IP) == nil {
		allErrs = append(allErrs, field.Invalid(fldPath, IP, fmt.Sprintf("invalid ip address: %s", IP)))
	}
	return allErrs
}

func ValidateObjectReference(ref *corev1.ObjectReference, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if ref.Kind == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("kind"), "must be set"))
	}
	if ref.APIVersion == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("apiVersion"), "must be set"))
	}
	if ref.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "must be set"))
	} else if errs := validation.IsDNS1123Subdomain(ref.Name); len(errs) > 0 { // length between 1 and 253 characters.
		allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), ref.Name, "must be a DNS-1123 subdomain"))
	}
	if ref.Namespace != "" {
		if errs := validation.IsDNS1123Label(ref.Namespace); len(errs) > 0 { // length between 1 and 63 characters.
			allErrs = append(allErrs, field.Invalid(fldPath.Child("namespace"), ref.Namespace, "must be a DNS-1123 label"))
		}
	}
	validation.IsDNS1123Label(ref.Namespace)

	// IsDNS1123Subdomain checks that a string is a valid DNS subdomain, which is defined as a string of length between 1 and 253 characters.
	validation.IsDNS1123Subdomain(ref.Name)
	return allErrs
}
