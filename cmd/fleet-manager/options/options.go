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

package options

import (
	"github.com/spf13/pflag"
)

type Options struct {
	MetricsBindAddr         string
	EnableLeaderElection    bool
	LeaderElectionNamespace string
	ProfilerAddress         string
	HealthAddr              string
	Concurrency             int
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

	fs.StringVar(
		&opt.HealthAddr,
		"health-addr",
		":9440",
		"The address the health endpoint binds to.",
	)
	fs.IntVar(
		&opt.Concurrency,
		"concurrency",
		5,
		"Number of Fleet API resources to process simultaneously",
	)
}
