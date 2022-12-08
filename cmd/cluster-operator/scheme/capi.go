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

// code in the package copied from: https://github.com/kubernetes-sigs/cluster-api/blob/v1.2.5/main.go
package scheme

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	clusterv1alpha4 "sigs.k8s.io/cluster-api/api/v1alpha4"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1alpha3 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	bootstrapv1alpha4 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha4"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	controlplanev1alpha3 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"
	controlplanev1alpha4 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha4"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	addonsv1alpha3 "sigs.k8s.io/cluster-api/exp/addons/api/v1alpha3"
	addonsv1alpha4 "sigs.k8s.io/cluster-api/exp/addons/api/v1alpha4"
	addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"
	expv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	expv1alpha4 "sigs.k8s.io/cluster-api/exp/api/v1alpha4"
	expv1 "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1alpha1"
	runtimev1 "sigs.k8s.io/cluster-api/exp/runtime/api/v1alpha1"
)

func init() {
	// capi core
	_ = clientgoscheme.AddToScheme(Scheme)
	_ = apiextensionsv1.AddToScheme(Scheme)
	_ = clusterv1alpha3.AddToScheme(Scheme)
	_ = clusterv1alpha4.AddToScheme(Scheme)
	_ = clusterv1.AddToScheme(Scheme)
	_ = expv1alpha3.AddToScheme(Scheme)
	_ = expv1alpha4.AddToScheme(Scheme)
	_ = expv1.AddToScheme(Scheme)
	_ = addonsv1alpha3.AddToScheme(Scheme)
	_ = addonsv1alpha4.AddToScheme(Scheme)
	_ = addonsv1.AddToScheme(Scheme)
	_ = runtimev1.AddToScheme(Scheme)
	_ = ipamv1.AddToScheme(Scheme)
	// bootstrap
	_ = bootstrapv1alpha3.AddToScheme(Scheme)
	_ = bootstrapv1alpha4.AddToScheme(Scheme)
	_ = bootstrapv1.AddToScheme(Scheme)
	// kubeadm controlplane
	_ = clientgoscheme.AddToScheme(Scheme)
	_ = clusterv1.AddToScheme(Scheme)
	_ = controlplanev1alpha3.AddToScheme(Scheme)
	_ = controlplanev1alpha4.AddToScheme(Scheme)
	_ = controlplanev1.AddToScheme(Scheme)
}
