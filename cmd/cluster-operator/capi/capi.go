/*
Copyright 2022 The Kubernetes Authors.

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

// code in the package copied from: https://github.com/kubernetes-sigs/cluster-api/blob/v1.2.5/main.go
package capi

import (
	"context"
	"fmt"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	kubeadmbootstrapcontrollers "sigs.k8s.io/cluster-api/bootstrap/kubeadm/controllers"
	"sigs.k8s.io/cluster-api/controllers"
	"sigs.k8s.io/cluster-api/controllers/remote"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	kubeadmcontrolplanecontrollers "sigs.k8s.io/cluster-api/controlplane/kubeadm/controllers"
	kcpwebhooks "sigs.k8s.io/cluster-api/controlplane/kubeadm/webhooks"
	"sigs.k8s.io/cluster-api/webhooks"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"kurator.dev/kurator/cmd/cluster-operator/config"
)

var (
	log = ctrl.Log.WithName("capi")
)

func InitControllers(ctx context.Context, opts *config.Options, mgr ctrl.Manager) error {
	if err := setupReconcilers(ctx, opts, mgr); err != nil {
		log.Error(err, "init capi reconcilers failed")
		return fmt.Errorf("init capi reconciler failed: %w", err)
	}

	if err := setupWebhooks(mgr); err != nil {
		log.Error(err, "init capi webhooks failed")
		return fmt.Errorf("init capi webhooks failed: %w", err)
	}

	return nil
}

func setupReconcilers(ctx context.Context, opts *config.Options, mgr ctrl.Manager) error {
	// Set up a ClusterCacheTracker and ClusterCacheReconciler to provide to controllers
	// requiring a connection to a remote cluster
	// ClusterCacheTracker.GetClient return client for remote cluster
	log := ctrl.Log.WithName("remote").WithName("ClusterCacheTracker")
	tracker, err := remote.NewClusterCacheTracker(
		mgr,
		remote.ClusterCacheTrackerOptions{
			Log:     &log,
			Indexes: remote.DefaultIndexes,
		},
	)
	if err != nil {
		return fmt.Errorf("unable to create cluster cache tracker, %w", err)
	}
	// ClusterCacheReconciler is responsible for stopping remote cluster caches when
	// the cluster for the remote cache is being deleted.
	if err := (&remote.ClusterCacheReconciler{
		Client:           mgr.GetClient(),
		Log:              ctrl.Log.WithName("remote").WithName("ClusterCacheReconciler"),
		Tracker:          tracker,
		WatchFilterValue: opts.WatchFilterValue,
	}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
		return fmt.Errorf("unable to create ClusterCache controller, %w", err)
	}

	if err := (&controllers.ClusterReconciler{
		Client:           mgr.GetClient(),
		APIReader:        mgr.GetAPIReader(),
		WatchFilterValue: opts.WatchFilterValue,
	}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
		return fmt.Errorf("unable to create Cluster controller, %w", err)
	}
	if err := (&controllers.MachineReconciler{
		Client:           mgr.GetClient(),
		APIReader:        mgr.GetAPIReader(),
		Tracker:          tracker,
		WatchFilterValue: opts.WatchFilterValue,
	}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
		return fmt.Errorf("unable to create Machine controller, %w", err)
	}
	if err := (&controllers.MachineSetReconciler{
		Client:           mgr.GetClient(),
		APIReader:        mgr.GetAPIReader(),
		Tracker:          tracker,
		WatchFilterValue: opts.WatchFilterValue,
	}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
		return fmt.Errorf("unable to create MachineSet controller, %w", err)
	}
	if err := (&controllers.MachineDeploymentReconciler{
		Client:           mgr.GetClient(),
		APIReader:        mgr.GetAPIReader(),
		WatchFilterValue: opts.WatchFilterValue,
	}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
		return fmt.Errorf("unable to create MachineDeployment controller, %w", err)
	}

	if err := (&controllers.MachineHealthCheckReconciler{
		Client:           mgr.GetClient(),
		Tracker:          tracker,
		WatchFilterValue: opts.WatchFilterValue,
	}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
		return fmt.Errorf("unable to create MachineHealthCheck controller, %w", err)
	}

	if err := (&kubeadmcontrolplanecontrollers.KubeadmControlPlaneReconciler{
		Client:           mgr.GetClient(),
		APIReader:        mgr.GetAPIReader(),
		Tracker:          tracker,
		WatchFilterValue: opts.WatchFilterValue,
	}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
		return fmt.Errorf("unable to create KubeadmControlPlane controller, %w", err)
	}

	if err := (&kubeadmbootstrapcontrollers.KubeadmConfigReconciler{
		Client:           mgr.GetClient(),
		WatchFilterValue: opts.WatchFilterValue,
	}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
		return fmt.Errorf("unable to create KubeadmConfig controller, %w", err)
	}

	return nil
}

func setupWebhooks(mgr ctrl.Manager) error {
	// NOTE: ClusterClass and managed topologies are behind ClusterTopology feature gate flag; the webhook
	// is going to prevent creating or updating new objects in case the feature flag is disabled.
	if err := (&webhooks.ClusterClass{Client: mgr.GetClient()}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create ClusterClass webhook, %w", err)
	}

	// NOTE: ClusterClass and managed topologies are behind ClusterTopology feature gate flag; the webhook
	// is going to prevent usage of Cluster.Topology in case the feature flag is disabled.
	if err := (&webhooks.Cluster{Client: mgr.GetClient()}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create Cluster webhook, %w", err)
	}

	if err := (&clusterv1.Machine{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create Machine webhook, %w", err)
	}

	if err := (&clusterv1.MachineSet{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create MachineSet webhook, %w", err)
	}

	if err := (&clusterv1.MachineDeployment{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create MachineDeployment webhook, %w", err)
	}

	if err := (&clusterv1.MachineHealthCheck{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create MachineHealthCheck webhook, %w", err)
	}

	if err := (&bootstrapv1.KubeadmConfig{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create KubeadmConfig webhook, %w", err)
	}
	if err := (&bootstrapv1.KubeadmConfigTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create KubeadmConfigTemplate webhook, %w", err)
	}

	if err := (&controlplanev1.KubeadmControlPlane{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create KubeadmControlPlane webhook, %w", err)
	}

	if err := (&kcpwebhooks.ScaleValidator{
		Client: mgr.GetClient(),
	}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create KubeadmControlPlane scale webhook, %w", err)
	}

	if err := (&controlplanev1.KubeadmControlPlaneTemplate{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create KubeadmControlPlaneTemplate webhook, %w", err)
	}

	return nil
}

func concurrency(c int) controller.Options {
	return controller.Options{MaxConcurrentReconciles: c}
}
