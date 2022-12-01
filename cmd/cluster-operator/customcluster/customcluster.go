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

	"kurator.dev/kurator/pkg/controllers"
)

var log = ctrl.Log.WithName("custom_cluster")

func InitControllers(ctx context.Context, mgr ctrl.Manager) error {
	if err := (&controllers.CustomClusterController{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "CustomCluster")
		return err
	}

	if err := (&controllers.CustomMachineController{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(ctx, mgr, controller.Options{}); err != nil {
		log.Error(err, "unable to create controller", "controller", "CustomMachine")
		return err
	}

	return nil
}
