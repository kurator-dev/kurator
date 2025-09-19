/*
Copyright 2022-2025 Kurator Authors.

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

package typemeta

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	awsinfrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
)

var (
	PropagationPolicy = metav1.TypeMeta{
		APIVersion: "policy.karmada.io/v1alpha1",
		Kind:       "PropagationPolicy",
	}

	ClusterPropagationPolicy = metav1.TypeMeta{
		APIVersion: "policy.karmada.io/v1alpha1",
		Kind:       "ClusterPropagationPolicy",
	}

	Secret = metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Secret",
	}

	Service = metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Service",
	}

	Endpoints = metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Endpoints",
	}

	OverridePolicy = metav1.TypeMeta{
		APIVersion: "policy.karmada.io/v1alpha1",
		Kind:       "OverridePolicy",
	}

	AWSClusterStaticIdentity = metav1.TypeMeta{
		APIVersion: awsinfrav1.GroupVersion.String(),
		Kind:       string(awsinfrav1.ClusterStaticIdentityKind),
	}
)
