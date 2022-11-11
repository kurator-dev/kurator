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
	"time"
)

var (
	// MaxEKSSyncPeriod is the maximum allowed duration for the sync-period flag when using EKS. It is set to 10 minutes
	// because during resync it will create a new AWS auth token which can a maximum life of 15 minutes and this ensures
	// the token (and kubeconfig secret) is refreshed before token expiration.
	MaxEKSSyncPeriod = time.Minute * 10
)

type AWSOptions struct {
	ClusterConcurrency       int
	InstanceStateConcurrency int
	MachineConcurrency       int
	ServiceEndpoints         string
	SyncPeriod               time.Duration
}
