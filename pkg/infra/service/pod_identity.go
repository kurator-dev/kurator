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

package service

type PodIdentity interface {
	// Reconcile ensures the resources for Pod Idenetity are created and ready to use.
	Reconcile() error
	// Delete ensures the resources for Pod Idenetity is deleted.
	Delete() error
	// ServiceAccountIssuer returns the service account issuer for the Pod Identity.
	ServiceAccountIssuer() string
}

var _ PodIdentity = &NopPodIdentity{}

type NopPodIdentity struct {
}

func (pi *NopPodIdentity) Reconcile() error {
	return nil
}

func (pi *NopPodIdentity) Delete() error {
	return nil
}

func (pi *NopPodIdentity) ServiceAccountIssuer() string {
	return ""
}
