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

package provider

import (
	"context"
)

type Provider interface {
	// Precheck returns error when there's wrong with provider configuration.
	Precheck(ctx context.Context) error
	// Reconcile ensures all resources used by Provider.
	Reconcile(ctx context.Context) error
	// Clean removes all resources created by the provider.
	Clean(ctx context.Context) error
	// IsInitialized returns true when kube apiserver is accessible.
	IsInitialized(ctx context.Context) error
	// IsReady returns true when the cluster is ready to be used.
	IsReady(ctx context.Context) error
}
