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
	eksbootstrapcontrollers "sigs.k8s.io/cluster-api-provider-aws/v2/bootstrap/eks/controllers"
	"sigs.k8s.io/cluster-api-provider-aws/v2/controllers"
	ekscontrolplanev1 "sigs.k8s.io/cluster-api-provider-aws/v2/controlplane/eks/api/v1beta2"
	ekscontrolplanecontrollers "sigs.k8s.io/cluster-api-provider-aws/v2/controlplane/eks/controllers"
	expinfrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/exp/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-aws/v2/exp/controlleridentitycreator"
	expcontrollers "sigs.k8s.io/cluster-api-provider-aws/v2/exp/controllers"
	"sigs.k8s.io/cluster-api-provider-aws/v2/exp/instancestate"
	"sigs.k8s.io/cluster-api-provider-aws/v2/feature"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/cloud/endpoints"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"kurator.dev/kurator/cmd/cluster-operator/config"
)

var (
	log                = ctrl.Log.WithName("aws")
	errEKSInvalidFlags = errors.New("invalid EKS flag combination")
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

	if feature.Gates.Enabled(feature.EKS) {
		log.Info("enabling EKS controllers")

		enableIAM := feature.Gates.Enabled(feature.EKSEnableIAM)
		allowAddRoles := feature.Gates.Enabled(feature.EKSAllowAddRoles)
		log.V(2).Info("EKS IAM role creation", "enabled", enableIAM)
		log.V(2).Info("EKS IAM additional roles", "enabled", allowAddRoles)
		if allowAddRoles && !enableIAM {
			log.Error(errEKSInvalidFlags, "cannot use EKSAllowAddRoles flag without EKSEnableIAM")
			return errors.New("cannot use EKSAllowAddRoles flag without EKSEnableIAM")
		}

		log.V(2).Info("enabling EKS control plane controller")
		if err := (&ekscontrolplanecontrollers.AWSManagedControlPlaneReconciler{
			Client:               mgr.GetClient(),
			EnableIAM:            enableIAM,
			AllowAdditionalRoles: allowAddRoles,
			Endpoints:            awsServiceEndpoints,
			WatchFilterValue:     opts.WatchFilterValue,
			ExternalResourceGC:   externalResourceGC,
		}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.ClusterConcurrency, RecoverPanic: true}); err != nil {
			log.Error(err, "unable to create controller", "controller", "AWSManagedControlPlane")
			return errors.New("unable to create AWSManagedControlPlane controlle")
		}

		log.V(2).Info("enabling EKS bootstrap controller")
		if err := (&eksbootstrapcontrollers.EKSConfigReconciler{
			Client:           mgr.GetClient(),
			WatchFilterValue: opts.WatchFilterValue,
		}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.ClusterConcurrency, RecoverPanic: true}); err != nil {
			log.Error(err, "unable to create controller", "controller", "EKSConfig")
			return errors.New("unable to create EKSConfig controlle")
		}

		if feature.Gates.Enabled(feature.EKSFargate) {
			log.V(2).Info("enabling EKS fargate profile controller")
			if err := (&expcontrollers.AWSFargateProfileReconciler{
				Client:           mgr.GetClient(),
				Recorder:         mgr.GetEventRecorderFor("awsfargateprofile-reconciler"),
				EnableIAM:        enableIAM,
				Endpoints:        awsServiceEndpoints,
				WatchFilterValue: opts.WatchFilterValue,
			}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.ClusterConcurrency, RecoverPanic: true}); err != nil {
				log.Error(err, "unable to create controller", "controller", "AWSFargateProfile")
				return errors.New("unable to create AWSFargateProfile controlle")
			}
		}

		if feature.Gates.Enabled(feature.MachinePool) {
			log.V(2).Info("enabling EKS managed machine pool controller")
			if err := (&expcontrollers.AWSManagedMachinePoolReconciler{
				AllowAdditionalRoles: allowAddRoles,
				Client:               mgr.GetClient(),
				EnableIAM:            enableIAM,
				Endpoints:            awsServiceEndpoints,
				Recorder:             mgr.GetEventRecorderFor("awsmanagedmachinepool-reconciler"),
				WatchFilterValue:     opts.WatchFilterValue,
			}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.InstanceStateConcurrency, RecoverPanic: true}); err != nil {
				log.Error(err, "unable to create controller", "controller", "AWSManagedMachinePool")
				return errors.New("unable to create AWSManagedMachinePool controlle")
			}
		}

		log.Info("enabling EKS webhooks")
		if err := (&ekscontrolplanev1.AWSManagedControlPlane{}).SetupWebhookWithManager(mgr); err != nil {
			log.Error(err, "unable to create webhook", "webhook", "AWSManagedControlPlane")
			return err
		}
		if feature.Gates.Enabled(feature.EKSFargate) {
			if err := (&expinfrav1.AWSFargateProfile{}).SetupWebhookWithManager(mgr); err != nil {
				log.Error(err, "unable to create webhook", "webhook", "AWSFargateProfile")
				return err
			}
		}
		if feature.Gates.Enabled(feature.MachinePool) {
			if err := (&expinfrav1.AWSManagedMachinePool{}).SetupWebhookWithManager(mgr); err != nil {
				log.Error(err, "unable to create webhook", "webhook", "AWSManagedMachinePool")
				return err
			}
		}
	}
	if feature.Gates.Enabled(feature.MachinePool) {
		log.V(2).Info("enabling machine pool controller")
		if err := (&expcontrollers.AWSMachinePoolReconciler{
			Client:           mgr.GetClient(),
			Recorder:         mgr.GetEventRecorderFor("awsmachinepool-controller"),
			WatchFilterValue: opts.WatchFilterValue,
		}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.InstanceStateConcurrency, RecoverPanic: true}); err != nil {
			log.Error(err, "unable to create controller", "controller", "AWSMachinePool")
			return errors.New("unable to create AWSMachinePool controlle")
		}

		log.Info("enabling webhook for AWSMachinePool")
		if err := (&expinfrav1.AWSMachinePool{}).SetupWebhookWithManager(mgr); err != nil {
			log.Error(err, "unable to create webhook", "webhook", "AWSMachinePool")
			return err
		}
	}
	if feature.Gates.Enabled(feature.EventBridgeInstanceState) {
		log.Info("EventBridge notifications enabled. enabling AWSInstanceStateController")
		if err := (&instancestate.AwsInstanceStateReconciler{
			Client:           mgr.GetClient(),
			Log:              ctrl.Log.WithName("controllers").WithName("AWSInstanceStateController"),
			Endpoints:        awsServiceEndpoints,
			WatchFilterValue: opts.WatchFilterValue,
		}).SetupWithManager(ctx, mgr, controller.Options{MaxConcurrentReconciles: opts.InstanceStateConcurrency, RecoverPanic: true}); err != nil {
			log.Error(err, "unable to create controller", "controller", "AWSInstanceStateController")
			return errors.New("unable to create AWSInstanceStateController controlle")
		}
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
	if feature.Gates.Enabled(feature.BootstrapFormatIgnition) {
		log.Info("Enabling Ignition support for machine bootstrap data")
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
