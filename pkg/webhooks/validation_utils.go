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
// A lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-',
// must start and end with an alphanumeric character,
// and must have a maximum length of 253 characters.
func validateDNS1123Domain(domain string, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if errs := validation.IsDNS1123Subdomain(domain); len(errs) > 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, domain, strings.Join(errs, "; ")))
	}
	return allErrs
}
