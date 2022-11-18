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
	"sigs.k8s.io/cluster-api/controllers"
	"sigs.k8s.io/cluster-api/controllers/remote"
	addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"
	addonscontrollers "sigs.k8s.io/cluster-api/exp/addons/controllers"
	expv1 "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	expcontrollers "sigs.k8s.io/cluster-api/exp/controllers"
	"sigs.k8s.io/cluster-api/feature"
	"sigs.k8s.io/cluster-api/webhooks"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	if err := (&remote.ClusterCacheReconciler{
		Client:           mgr.GetClient(),
		Log:              ctrl.Log.WithName("remote").WithName("ClusterCacheReconciler"),
		Tracker:          tracker,
		WatchFilterValue: opts.WatchFilterValue,
	}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
		return fmt.Errorf("unable to create ClusterCache controller, %w", err)
	}

	if feature.Gates.Enabled(feature.ClusterTopology) {
		unstructuredCachingClient, err := client.NewDelegatingClient(
			client.NewDelegatingClientInput{
				// Use the default client for write operations.
				Client: mgr.GetClient(),
				// For read operations, use the same cache used by all the controllers but ensure
				// unstructured objects will be also cached (this does not happen with the default client).
				CacheReader:       mgr.GetCache(),
				CacheUnstructured: true,
			},
		)
		if err != nil {
			return fmt.Errorf("unable to create unstructured caching client for ClusterTopology, %w", err)
		}

		if err := (&controllers.ClusterClassReconciler{
			Client:                    mgr.GetClient(),
			APIReader:                 mgr.GetAPIReader(),
			UnstructuredCachingClient: unstructuredCachingClient,
			WatchFilterValue:          opts.WatchFilterValue,
		}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
			return fmt.Errorf("unable to create ClusterClass controller, %w", err)
		}

		if err := (&controllers.ClusterTopologyReconciler{
			Client:                    mgr.GetClient(),
			APIReader:                 mgr.GetAPIReader(),
			UnstructuredCachingClient: unstructuredCachingClient,
			WatchFilterValue:          opts.WatchFilterValue,
		}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
			return fmt.Errorf("unable to create ClusterTopology controller, %w", err)
		}

		if err := (&controllers.MachineDeploymentTopologyReconciler{
			Client:           mgr.GetClient(),
			APIReader:        mgr.GetAPIReader(),
			WatchFilterValue: opts.WatchFilterValue,
		}).SetupWithManager(ctx, mgr, controller.Options{}); err != nil {
			return fmt.Errorf("unable to create MachineDeploymentTopology controller, %w", err)
		}

		if err := (&controllers.MachineSetTopologyReconciler{
			Client:           mgr.GetClient(),
			APIReader:        mgr.GetAPIReader(),
			WatchFilterValue: opts.WatchFilterValue,
		}).SetupWithManager(ctx, mgr, controller.Options{}); err != nil {
			return fmt.Errorf("unable to create MachineSetTopology controller, %w", err)
		}
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

	if feature.Gates.Enabled(feature.MachinePool) {
		if err := (&expcontrollers.MachinePoolReconciler{
			Client:           mgr.GetClient(),
			WatchFilterValue: opts.WatchFilterValue,
		}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
			return fmt.Errorf("unable to create MachinePool controller, %w", err)
		}
	}

	if feature.Gates.Enabled(feature.ClusterResourceSet) {
		if err := (&addonscontrollers.ClusterResourceSetReconciler{
			Client:           mgr.GetClient(),
			Tracker:          tracker,
			WatchFilterValue: opts.WatchFilterValue,
		}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
			return fmt.Errorf("unable to create ClusterResourceSet controller, %w", err)
		}
		if err := (&addonscontrollers.ClusterResourceSetBindingReconciler{
			Client:           mgr.GetClient(),
			WatchFilterValue: opts.WatchFilterValue,
		}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
			return fmt.Errorf("unable to create ClusterResourceSetBinding controller, %w", err)
		}
	}

	if err := (&controllers.MachineHealthCheckReconciler{
		Client:           mgr.GetClient(),
		Tracker:          tracker,
		WatchFilterValue: opts.WatchFilterValue,
	}).SetupWithManager(ctx, mgr, concurrency(opts.Concurrency)); err != nil {
		return fmt.Errorf("unable to create MachineHealthCheck controller, %w", err)
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

	if feature.Gates.Enabled(feature.MachinePool) {
		if err := (&expv1.MachinePool{}).SetupWebhookWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create MachinePool webhook, %w", err)
		}
	}

	if feature.Gates.Enabled(feature.ClusterResourceSet) {
		if err := (&addonsv1.ClusterResourceSet{}).SetupWebhookWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create ClusterResourceSet webhook, %w", err)
		}
	}

	if err := (&clusterv1.MachineHealthCheck{}).SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create MachineHealthCheck webhook, %w", err)
	}

	return nil
}

func concurrency(c int) controller.Options {
	return controller.Options{MaxConcurrentReconciles: c}
}
