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

package infra

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"kurator.dev/kurator/cmd/cluster-operator/options"
	"kurator.dev/kurator/pkg/controllers"
	"kurator.dev/kurator/pkg/webhooks"
)

var log = ctrl.Log.WithName("infra cluster")

func InitControllers(ctx context.Context, opts *options.Options, mgr ctrl.Manager) error {
	if err := (&controllers.ClusterController{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		RequeueAfter: opts.RequeueAfter,
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "Infra Cluster")
		return err
	}

	if err := (&webhooks.ClusterWebhook{
		Client: mgr.GetClient(),
	}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create Cluster webhook, %w", err)
	}

	return nil
}
