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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	cgrecord "k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/cluster-api-provider-aws/v2/feature"
	"sigs.k8s.io/cluster-api/util/record"
	ctrl "sigs.k8s.io/controller-runtime"

	"kurator.dev/kurator/cmd/cluster-operator/aws"
	"kurator.dev/kurator/cmd/cluster-operator/capi"
	"kurator.dev/kurator/cmd/cluster-operator/config"
	"kurator.dev/kurator/cmd/cluster-operator/customcluster"
	"kurator.dev/kurator/cmd/cluster-operator/scheme"
	"kurator.dev/kurator/pkg/version"
)

var log = ctrl.Log.WithName("cluster-operator")

func main() {
	klog.InitFlags(nil)
	rand.Seed(time.Now().UnixNano())
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	cmd := newRootCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Println("execute kurator command failed: ", err)
		os.Exit(-1)
	}
}

func newRootCommand() *cobra.Command {
	opts := &config.Options{}
	cmd := &cobra.Command{
		Use:          "cluster-operator",
		Short:        "Kurator builds distributed cloud-native stacks.",
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.SyncPeriod > config.MaxEKSSyncPeriod {
				return fmt.Errorf("syn-period(%v) should not greater than EKS max-sync-period %v", opts.SyncPeriod, config.MaxEKSSyncPeriod)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctrl.SetLogger(klogr.New())
			ctx := ctrl.SetupSignalHandler()
			return run(ctx, opts)
		},
	}
	cmd.AddCommand(newVersionCommand())

	cmd.ResetFlags()

	opts.AddFlags(cmd.PersistentFlags())

	return cmd
}

func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of kurator cluster-operator",
		RunE: func(cmd *cobra.Command, args []string) error {
			v := version.Get()
			y, err := json.MarshalIndent(&v, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(y))
			return nil
		},
	}
	return cmd
}

func run(ctx context.Context, opts *config.Options) error {
	if opts.WatchNamespace != "" {
		log.Info("Watching cluster-api objects only in namespace for reconciliation", "namespace", opts.WatchNamespace)
	}

	if opts.ProfilerAddress != "" {
		log.Info("Profiler listening for requests", "profiler-address", opts.ProfilerAddress)
		go func() {
			log.Error(http.ListenAndServe(opts.ProfilerAddress, nil), "listen and serve error")
		}()
	}

	// Machine and cluster operations can create enough events to trigger the event recorder spam filter
	// Setting the burst size higher ensures all events will be recorded and submitted to the API
	broadcaster := cgrecord.NewBroadcasterWithCorrelatorOptions(cgrecord.CorrelatorOptions{
		BurstSize: 100,
	})

	restConfig := ctrl.GetConfigOrDie()
	restConfig.UserAgent = "kurator-cluster-operator"
	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme:                     scheme.Scheme,
		MetricsBindAddress:         opts.MetricsBindAddr,
		LeaderElection:             opts.EnableLeaderElection,
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaderElectionID:           "kurator-cluster-operator-leader-elect",
		LeaderElectionNamespace:    opts.LeaderElectionNamespace,
		SyncPeriod:                 &opts.SyncPeriod,
		Namespace:                  opts.WatchNamespace,
		EventBroadcaster:           broadcaster,
		Port:                       opts.WebhookPort,
		CertDir:                    opts.WebhookCertDir,
		HealthProbeBindAddress:     opts.HealthAddr,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		return err
	}

	record.InitFromRecorder(mgr.GetEventRecorderFor("cluster-operator"))
	log.V(1).Info(fmt.Sprintf("feature gates: %+v\n", feature.Gates))

	// capi
	if err = capi.InitControllers(ctx, opts, mgr); err != nil {
		return fmt.Errorf("capi init fail, %w", err)
	}

	// capa
	if err = aws.InitControllers(ctx, opts, mgr); err != nil {
		return fmt.Errorf("capa init fail, %w", err)
	}

	if err = customcluster.InitControllers(ctx, mgr); err != nil {
		return err
	}

	if err := mgr.AddReadyzCheck("webhook", mgr.GetWebhookServer().StartedChecker()); err != nil {
		log.Error(err, "unable to create ready check")
		return err
	}

	if err := mgr.AddHealthzCheck("webhook", mgr.GetWebhookServer().StartedChecker()); err != nil {
		log.Error(err, "unable to create health check")
		return err
	}

	log.Info("starting manager", "version", version.Get().String())
	if err := mgr.Start(ctx); err != nil {
		log.Error(err, "running manager error")
		return err
	}

	return nil
}
