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
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	cgrecord "k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/cluster-api/util/record"
	ctrl "sigs.k8s.io/controller-runtime"

	"kurator.dev/kurator/cmd/fleet-manager/application"
	"kurator.dev/kurator/cmd/fleet-manager/options"
	"kurator.dev/kurator/cmd/fleet-manager/scheme"
	fleet "kurator.dev/kurator/pkg/fleet-manager"
	"kurator.dev/kurator/pkg/fleet-manager/manifests"
	"kurator.dev/kurator/pkg/util"
	"kurator.dev/kurator/pkg/version"
)

var log = ctrl.Log.WithName("fleet-manager")

func main() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	cmd := newRootCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Println("execute fleet-manager command failed: ", err)
		os.Exit(-1)
	}
}

func newRootCommand() *cobra.Command {
	opts := &options.Options{}
	cmd := &cobra.Command{
		Use:          "fleet-manager",
		Short:        "fleet-manager is in charge of fleet management.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctrl.SetLogger(klogr.New())
			util.PrintFlags(log, cmd.Flags())
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
		Short: "Print the version of kurator fleet-manager",
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

func run(ctx context.Context, opts *options.Options) error {
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
	restConfig.UserAgent = "kurator-fleet-manager"
	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme:                     scheme.Scheme,
		MetricsBindAddress:         opts.MetricsBindAddr,
		LeaderElection:             opts.EnableLeaderElection,
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaderElectionID:           "kurator-fleet-manager-leader-elect",
		LeaderElectionNamespace:    opts.LeaderElectionNamespace,
		EventBroadcaster:           broadcaster,
	})
	if err != nil {
		log.Error(err, "unable to create manager")
		return err
	}

	record.InitFromRecorder(mgr.GetEventRecorderFor("cluster-operator"))
	if err := (&fleet.FleetManager{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Manifests: manifests.BuiltinOrDir(opts.ManifestsDir),
	}).SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to set up fleet manager", "controller", "fleet manager")
		return err
	}

	if err = application.InitControllers(ctx, opts, mgr); err != nil {
		return fmt.Errorf("application init fail, %w", err)
	}

	log.Info("starting manager", "version", version.Get().String())
	if err := mgr.Start(ctx); err != nil {
		log.Error(err, "running manager error")
		return err
	}

	return nil
}
