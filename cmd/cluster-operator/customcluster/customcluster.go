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

package customcluster

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"kurator.dev/kurator/cmd/cluster-operator/options"
	clusteroperator "kurator.dev/kurator/pkg/cluster-operator"
	"kurator.dev/kurator/pkg/webhooks"
)

var log = ctrl.Log.WithName("custom_cluster")

func InitControllers(ctx context.Context, opts *options.Options, mgr ctrl.Manager) error {
	if err := (&clusteroperator.CustomClusterController{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		APIReader: mgr.GetAPIReader(),
	}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.Concurrency, RecoverPanic: true}); err != nil {
		log.Error(err, "unable to create controller", "controller", "CustomCluster")
		return err
	}

	if err := (&clusteroperator.CustomMachineController{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		APIReader: mgr.GetAPIReader(),
	}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.Concurrency, RecoverPanic: true}); err != nil {
		log.Error(err, "unable to create controller", "controller", "CustomMachine")
		return err
	}

	if err := (&webhooks.CustomClusterWebhook{
		Client: mgr.GetClient(),
	}).SetupWebhookWithManager(mgr); err != nil {
		log.Error(err, "unable to create CustomCluster webhook", "Webhook", "CustomCluster")
		return err
	}

	return nil
}
