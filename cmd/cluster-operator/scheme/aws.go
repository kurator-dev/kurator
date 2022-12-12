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
package scheme

import (
	cgscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/component-base/logs/json/register"
	infrav1beta1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta1"
	infrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	eksbootstrapv1beta1 "sigs.k8s.io/cluster-api-provider-aws/v2/bootstrap/eks/api/v1beta1"
	eksbootstrapv1 "sigs.k8s.io/cluster-api-provider-aws/v2/bootstrap/eks/api/v1beta2"
	ekscontrolplanev1beta1 "sigs.k8s.io/cluster-api-provider-aws/v2/controlplane/eks/api/v1beta1"
	ekscontrolplanev1 "sigs.k8s.io/cluster-api-provider-aws/v2/controlplane/eks/api/v1beta2"
	expinfrav1beta1 "sigs.k8s.io/cluster-api-provider-aws/v2/exp/api/v1beta1"
	expinfrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/exp/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	expclusterv1 "sigs.k8s.io/cluster-api/exp/api/v1beta1"
)

func init() {
	_ = eksbootstrapv1.AddToScheme(Scheme)
	_ = eksbootstrapv1beta1.AddToScheme(Scheme)
	_ = cgscheme.AddToScheme(Scheme)
	_ = clusterv1.AddToScheme(Scheme)
	_ = expclusterv1.AddToScheme(Scheme)
	_ = ekscontrolplanev1.AddToScheme(Scheme)
	_ = ekscontrolplanev1beta1.AddToScheme(Scheme)
	_ = infrav1.AddToScheme(Scheme)
	_ = infrav1beta1.AddToScheme(Scheme)
	_ = expinfrav1beta1.AddToScheme(Scheme)
	_ = expinfrav1.AddToScheme(Scheme)
}
