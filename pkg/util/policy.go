package util

import (
	"context"
	"fmt"

	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	karmadaclientset "github.com/karmada-io/karmada/pkg/generated/clientset/versioned"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

// resource like Namespace will be propagated by default, do not apply in ResourceSelector
var ignoredResources = map[schema.GroupVersionKind]struct{}{
	{
		Group:   "",
		Version: "v1",
		Kind:    "Namespace",
	}: {},
}

func AppendResourceSelector(discoveryClient discovery.DiscoveryInterface,
	cpp *policyv1alpha1.ClusterPropagationPolicy,
	pp *policyv1alpha1.PropagationPolicy,
	resourceList kube.ResourceList) error {
	_, lists, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		return err
	}

	namespacedResources := map[schema.GroupVersionKind]struct{}{}
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			if resource.Namespaced {
				gvk := schema.GroupVersionKind{
					Group:   gv.Group,
					Version: gv.Version,
					Kind:    resource.Kind,
				}
				namespacedResources[gvk] = struct{}{}
				continue
			}
		}
	}

	for _, r := range resourceList {
		gvk := r.Mapping.GroupVersionKind
		if _, ok := ignoredResources[gvk]; ok {
			continue
		}

		gv := r.Mapping.GroupVersionKind.GroupVersion()
		gk := r.Mapping.GroupVersionKind.GroupKind()

		if _, ok := namespacedResources[gvk]; ok {
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

	return nil
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

func CreatePropagationPolicy(karmadaclient karmadaclientset.Interface, clusters []string, obj runtime.Object) error {
	pp, err := generatePropagationPolicy(clusters, obj)
	if err != nil {
		return fmt.Errorf("failed to generator propagation policy %w", err)
	}
	if _, err := karmadaclient.PolicyV1alpha1().PropagationPolicies(pp.Namespace).
		Create(context.TODO(), pp, metav1.CreateOptions{}); err != nil {
		return err
	}
	return nil
}
