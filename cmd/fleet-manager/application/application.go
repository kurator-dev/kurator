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

package application

import (
	"context"

	"istio.io/istio/pkg/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"kurator.dev/kurator/cmd/fleet-manager/options"
	"kurator.dev/kurator/pkg/fleet-manager"
	"kurator.dev/kurator/pkg/webhooks"
)

var log = ctrl.Log.WithName("application")

func InitControllers(ctx context.Context, opts *options.Options, mgr ctrl.Manager) error {
	if err := (&fleet.ApplicationManager{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.Concurrency, RecoverPanic: ptr.Of[bool](true)}); err != nil {
		log.Error(err, "unable to create controller", "controller", "Application")
		return err
	}

	if err := (&webhooks.ApplicationWebhook{
		Client: mgr.GetClient(),
	}).SetupWebhookWithManager(mgr); err != nil {
		log.Error(err, "unable to create Application webhook", "Webhook", "Application")
		return err
	}

	return nil
}
