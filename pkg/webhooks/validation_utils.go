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

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func validateIP(IP string, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if net.ParseIP(IP) == nil {
		allErrs = append(allErrs, field.Invalid(fldPath, IP, fmt.Sprintf("ip: %v is not a valid textual representation of an IP address", IP)))
	}
	return allErrs
}
