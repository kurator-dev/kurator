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

package util

import (
	"fmt"

	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/typemeta"
)

// resource like Namespace will be propagated by default, do not apply in ResourceSelector
var ignoredResources = map[schema.GroupVersionKind]struct{}{
	{
		Group:   "",
		Version: "v1",
		Kind:    "Namespace",
	}: {},
}

func AppendResourceSelector(cpp *policyv1alpha1.ClusterPropagationPolicy,
	pp *policyv1alpha1.PropagationPolicy,
	resourceList kube.ResourceList) {
	for _, r := range resourceList {
		gvk := r.Mapping.GroupVersionKind
		if _, ok := ignoredResources[gvk]; ok {
			continue
		}

		gv := r.Mapping.GroupVersionKind.GroupVersion()
		gk := r.Mapping.GroupVersionKind.GroupKind()

		if r.Namespaced() {
			s := policyv1alpha1.ResourceSelector{
				APIVersion: gv.String(),
				Kind:       gk.Kind,
				Name:       r.Name,
				Namespace:  r.Namespace,
			}
			pp.Spec.ResourceSelectors = append(pp.Spec.ResourceSelectors, s)
		} else {
			s := policyv1alpha1.ResourceSelector{
				APIVersion: gv.String(),
				Kind:       gk.Kind,
				Name:       r.Name,
			}
			cpp.Spec.ResourceSelectors = append(cpp.Spec.ResourceSelectors, s)
		}
	}
}

func generatePropagationPolicy(clusters []string, obj runtime.Object) (*policyv1alpha1.PropagationPolicy, error) {
	metaInfo, err := meta.Accessor(obj)
	if err != nil { // should not happen
		return nil, fmt.Errorf("object has no meta: %v", err)
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.GroupVersion().String() == "" || gvk.Kind == "" {
		return nil, fmt.Errorf("object miss type meta")
	}

	pp := &policyv1alpha1.PropagationPolicy{
		TypeMeta: typemeta.PropagationPolicy,
		ObjectMeta: metav1.ObjectMeta{
			Name:      metaInfo.GetName(),
			Namespace: metaInfo.GetNamespace(),
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
					Name:       metaInfo.GetName(),
					Namespace:  metaInfo.GetNamespace(),
				},
			},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: clusters,
				},
			},
		},
	}

	return pp, nil
}

func ApplyPropagationPolicy(c *client.Client, clusters []string, obj runtime.Object) error {
	pp, err := generatePropagationPolicy(clusters, obj)
	if err != nil {
		return fmt.Errorf("failed to generator propagation policy %w", err)
	}

	if err := c.UpdateResource(pp); err != nil {
		return fmt.Errorf("failed to apply propagation policy %w", err)
	}
	return nil
}
