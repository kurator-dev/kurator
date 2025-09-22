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

package thanos

import (
	"fmt"
	"os"

	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kurator.dev/kurator/pkg/typemeta"
)

var thanosSidecarRemoteService = &corev1.Service{
	TypeMeta: typemeta.Service,
	ObjectMeta: metav1.ObjectMeta{
		Name:      "thanos-sidecar-remote",
		Namespace: "thanos",
	},
	Spec: corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name:     "grpc",
				Protocol: corev1.ProtocolTCP,
				Port:     10901,
			},
			{
				Name:     "http",
				Protocol: corev1.ProtocolTCP,
				Port:     10902,
			},
		},
		ClusterIP: "None",
	},
}

func objectStoreSecret(filename string) (*corev1.Secret, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read object store config fail, %w", err)
	}

	return &corev1.Secret{
		// we need TypeMeta to create PropagationPolicy
		TypeMeta: typemeta.Secret,
		ObjectMeta: metav1.ObjectMeta{
			Name: "thanos-objstore-config",
		},
		StringData: map[string]string{
			"thanos.yaml": string(b),
		},
	}, nil
}

func thanosSidecarRemoteEndpoints(sidecarElbIPs []string) *corev1.Endpoints {
	addressList := make([]corev1.EndpointAddress, 0, len(sidecarElbIPs))
	for _, ip := range sidecarElbIPs {
		addressList = append(addressList, corev1.EndpointAddress{
			IP: ip,
		})
	}

	return &corev1.Endpoints{
		TypeMeta: typemeta.Endpoints,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thanos-sidecar-remote",
			Namespace: "thanos",
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: addressList,
				Ports: []corev1.EndpointPort{
					{
						Name:     "grpc",
						Protocol: corev1.ProtocolTCP,
						Port:     10901,
					},
					{
						Name:     "http",
						Protocol: corev1.ProtocolTCP,
						Port:     10902,
					},
				},
			},
		},
	}
}

func clusterOverridePolicy(cluster string) *policyv1alpha1.OverridePolicy {
	return &policyv1alpha1.OverridePolicy{
		TypeMeta: typemeta.OverridePolicy,
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusCRName + "-" + cluster,
			Namespace: monitoringNamespace,
		},
		Spec: policyv1alpha1.OverrideSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: prometheusGVK.GroupVersion().String(),
					Kind:       prometheusGVK.Kind,
					Name:       prometheusCRName,
				},
			},
			OverrideRules: []policyv1alpha1.RuleWithCluster{
				{
					Overriders: policyv1alpha1.Overriders{
						Plaintext: []policyv1alpha1.PlaintextOverrider{
							{
								Path:     "/spec/externalLabels",
								Operator: policyv1alpha1.OverriderOpAdd,
								Value:    apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("{ \"%s\": \"%s\"}", "cluster", cluster))},
							},
						},
					},
					TargetCluster: &policyv1alpha1.ClusterAffinity{
						ClusterNames: []string{cluster},
					},
				},
			},
		},
	}
}
