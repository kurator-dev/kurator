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

// code in the package copied from: https://github.com/kubernetes-sigs/cluster-api-provider-aws/blob/v1.5.1/main.go
package config

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
	"sigs.k8s.io/cluster-api-provider-aws/feature"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type Options struct {
	AWSOptions

	MetricsBindAddr         string
	EnableLeaderElection    bool
	LeaderElectionNamespace string
	WatchNamespace          string
	WatchFilterValue        string
	ProfilerAddress         string
	WebhookPort             int
	WebhookCertDir          string
	HealthAddr              string
}

func (opt *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&opt.MetricsBindAddr,
		"metrics-bind-addr",
		"localhost:8080",
		"The address the metric endpoint binds to.",
	)

	fs.BoolVar(
		&opt.EnableLeaderElection,
		"leader-elect",
		false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.",
	)

	fs.StringVar(
		&opt.WatchNamespace,
		"namespace",
		"",
		"Namespace that the controller watches to reconcile cluster-api objects. If unspecified, the controller watches for cluster-api objects across all namespaces.",
	)

	fs.StringVar(
		&opt.LeaderElectionNamespace,
		"leader-elect-namespace",
		"",
		"Namespace that the controller performs leader election in. If unspecified, the controller will discover which namespace it is running in.",
	)

	fs.StringVar(
		&opt.ProfilerAddress,
		"profiler-address",
		"",
		"Bind address to expose the pprof profiler (e.g. localhost:6060)",
	)

	fs.IntVar(
		&opt.ClusterConcurrency,
		"awscluster-concurrency",
		5,
		"Number of AWSClusters to process simultaneously",
	)

	fs.IntVar(
		&opt.InstanceStateConcurrency,
		"instance-state-concurrency",
		5,
		"Number of concurrent watches for instance state changes",
	)

	fs.IntVar(
		&opt.MachineConcurrency,
		"awsmachine-concurrency",
		10,
		"Number of AWSMachines to process simultaneously",
	)

	fs.DurationVar(
		&opt.SyncPeriod,
		"sync-period",
		10*time.Minute,
		fmt.Sprintf("The minimum interval at which watched resources are reconciled. If EKS is enabled the maximum allowed is %s", MaxEKSSyncPeriod),
	)

	fs.IntVar(
		&opt.WebhookPort,
		"webhook-port",
		9443,
		"Webhook Server port.",
	)

	fs.StringVar(
		&opt.WebhookCertDir,
		"webhook-cert-dir",
		"/tmp/k8s-webhook-server/serving-certs/",
		"Webhook cert dir, only used when webhook-port is specified.")

	fs.StringVar(
		&opt.HealthAddr,
		"health-addr",
		":9440",
		"The address the health endpoint binds to.",
	)

	fs.StringVar(
		&opt.ServiceEndpoints,
		"service-endpoints",
		"",
		"Set custom AWS service endpoins in semi-colon separated format: ${SigningRegion1}:${ServiceID1}=${URL},${ServiceID2}=${URL};${SigningRegion2}...",
	)

	fs.StringVar(
		&opt.WatchFilterValue,
		"watch-filter",
		"",
		fmt.Sprintf("Label value that the controller watches to reconcile cluster-api objects. Label key is always %s. If unspecified, the controller watches for all cluster-api objects.", clusterv1.WatchLabel),
	)

	feature.MutableGates.AddFlag(fs)
}
