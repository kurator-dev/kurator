/*
Copyright 2018 The Kubernetes Authors.

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

// code in the package copied from: https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/v2.0.0/main.go
package aws

import (
	"context"
	"errors"

	infrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-aws/v2/controllers"
	"sigs.k8s.io/cluster-api-provider-aws/v2/exp/controlleridentitycreator"
	"sigs.k8s.io/cluster-api-provider-aws/v2/feature"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/cloud/endpoints"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"kurator.dev/kurator/cmd/cluster-operator/config"
)

var (
	log = ctrl.Log.WithName("aws")
)

func InitControllers(ctx context.Context, opts *config.Options, mgr ctrl.Manager) error {
	externalResourceGC := false
	if feature.Gates.Enabled(feature.ExternalResourceGC) {
		log.Info("enabling external resource garbage collection")
		externalResourceGC = true
	}

	// Parse service endpoints.
	awsServiceEndpoints, err := endpoints.ParseFlag(opts.ServiceEndpoints)
	if err != nil {
		log.Error(err, "unable to parse service endpoints", "controller", "AWSCluster")
		return err
	}

	if err := (&controllers.AWSMachineReconciler{
		Client:           mgr.GetClient(),
		Log:              ctrl.Log.WithName("controllers").WithName("AWSMachine"),
		Recorder:         mgr.GetEventRecorderFor("awsmachine-controller"),
		Endpoints:        awsServiceEndpoints,
		WatchFilterValue: opts.WatchFilterValue,
	}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.MachineConcurrency, RecoverPanic: true}); err != nil {
		log.Error(err, "unable to create controller", "controller", "AWSMachine")
		return err
	}
	if err := (&controllers.AWSClusterReconciler{
		Client:             mgr.GetClient(),
		Recorder:           mgr.GetEventRecorderFor("awscluster-controller"),
		Endpoints:          awsServiceEndpoints,
		WatchFilterValue:   opts.WatchFilterValue,
		ExternalResourceGC: externalResourceGC,
	}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.ClusterConcurrency, RecoverPanic: true}); err != nil {
		log.Error(err, "unable to create controller", "controller", "AWSCluster")
		return err
	}

	if feature.Gates.Enabled(feature.AutoControllerIdentityCreator) {
		log.Info("AutoControllerIdentityCreator enabled")
		if err := (&controlleridentitycreator.AWSControllerIdentityReconciler{
			Client:           mgr.GetClient(),
			Log:              ctrl.Log.WithName("controllers").WithName("AWSControllerIdentity"),
			Endpoints:        awsServiceEndpoints,
			WatchFilterValue: opts.WatchFilterValue,
		}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.ClusterConcurrency, RecoverPanic: true}); err != nil {
			log.Error(err, "unable to create controller", "controller", "AWSControllerIdentity")
			return errors.New("unable to create AWSControllerIdentity controlle")
		}
	}

	if err := (&infrav1.AWSMachineTemplateWebhook{}).SetupWebhookWithManager(mgr); err != nil {
		log.Error(err, "unable to create webhook", "webhook", "AWSCluster")
		return err
	}
	if err := (&infrav1.AWSCluster{}).SetupWebhookWithManager(mgr); err != nil {
		log.Error(err, "unable to create webhook", "webhook", "AWSCluster")
		return err
	}
	if err := (&infrav1.AWSClusterTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		log.Error(err, "unable to create webhook", "webhook", "AWSClusterTemplate")
		return err
	}
	if err := (&infrav1.AWSClusterControllerIdentity{}).SetupWebhookWithManager(mgr); err != nil {
		log.Error(err, "unable to create webhook", "webhook", "AWSClusterControllerIdentity")
		return err
	}
	if err := (&infrav1.AWSClusterRoleIdentity{}).SetupWebhookWithManager(mgr); err != nil {
		log.Error(err, "unable to create webhook", "webhook", "AWSClusterRoleIdentity")
		return err
	}
	if err := (&infrav1.AWSClusterStaticIdentity{}).SetupWebhookWithManager(mgr); err != nil {
		log.Error(err, "unable to create webhook", "webhook", "AWSClusterStaticIdentity")
		return err
	}
	if err := (&infrav1.AWSMachine{}).SetupWebhookWithManager(mgr); err != nil {
		log.Error(err, "unable to create webhook", "webhook", "AWSMachine")
		return err
	}
	return nil
}
