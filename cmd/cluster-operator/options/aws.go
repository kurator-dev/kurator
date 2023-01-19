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
package options

import (
	"github.com/spf13/pflag"
)

type AWSOptions struct {
	ServiceEndpoints string
}

func (opt *AWSOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&opt.ServiceEndpoints,
		"service-endpoints",
		"",
		"Set custom AWS service endpoins in semi-colon separated format: ${SigningRegion1}:${ServiceID1}=${URL},${ServiceID2}=${URL};${SigningRegion2}...",
	)
}
