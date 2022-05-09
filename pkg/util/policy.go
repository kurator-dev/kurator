package util

import (
	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	"k8s.io/cli-runtime/pkg/resource"
)

// TODO: refactor me with istio plugin
func AppendClusterPropagationPolicy(cpp *policyv1alpha1.ClusterPropagationPolicy, r *resource.Info) {
	gv := r.Mapping.GroupVersionKind.GroupVersion()
	gk := r.Mapping.GroupVersionKind.GroupKind()
	s := policyv1alpha1.ResourceSelector{
		APIVersion: gv.String(),
		Kind:       gk.Kind,
		Name:       r.Name,
	}

	cpp.Spec.ResourceSelectors = append(cpp.Spec.ResourceSelectors, s)
}

// TODO: refactor me with istio plugin
func AppendPropagationPolicy(pp *policyv1alpha1.PropagationPolicy, r *resource.Info) {
	gv := r.Mapping.GroupVersionKind.GroupVersion()
	gk := r.Mapping.GroupVersionKind.GroupKind()
	s := policyv1alpha1.ResourceSelector{
		APIVersion: gv.String(),
		Kind:       gk.Kind,
		Name:       r.Name,
		Namespace:  r.Namespace,
	}

	pp.Spec.ResourceSelectors = append(pp.Spec.ResourceSelectors, s)
}
