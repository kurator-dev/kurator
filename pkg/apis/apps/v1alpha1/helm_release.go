/*
Copyright 2020 The Flux authors

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

package v1alpha1

import (
	helmv2b1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/fluxcd/pkg/apis/meta"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Note: copied from https://github.com/fluxcd/helm-controller/blob/main/api/v2beta1/helmrelease_types.go
// HelmRelease defines the desired state of a Helm release.
type HelmRelease struct {
	// Chart defines the template of the v1beta2.HelmChart that should be created
	// for this HelmRelease.
	// +required
	Chart HelmChartTemplate `json:"chart"`

	// Interval at which to reconcile the Helm release.
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^([0-9]+(\\.[0-9]+)?(ms|s|m|h))+$"
	// +required
	Interval metav1.Duration `json:"interval"`

	// Suspend tells the controller to suspend reconciliation for this HelmRelease,
	// it does not apply to already started reconciliations. Defaults to false.
	// +optional
	Suspend bool `json:"suspend,omitempty"`

	// ReleaseName used for the Helm release. Defaults to a composition of
	// '[TargetNamespace-]Name'.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=53
	// +kubebuilder:validation:Optional
	// +optional
	ReleaseName string `json:"releaseName,omitempty"`

	// TargetNamespace to target when performing operations for the HelmRelease.
	// Defaults to the namespace of the HelmRelease.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Optional
	// +optional
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// DependsOn may contain a meta.NamespacedObjectReference slice with
	// references to HelmRelease resources that must be ready before this HelmRelease
	// can be reconciled.
	// +optional
	DependsOn []meta.NamespacedObjectReference `json:"dependsOn,omitempty"`

	// Timeout is the time to wait for any individual Kubernetes operation (like Jobs
	// for hooks) during the performance of a Helm action. Defaults to '5m0s'.
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^([0-9]+(\\.[0-9]+)?(ms|s|m|h))+$"
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// MaxHistory is the number of revisions saved by Helm for this HelmRelease.
	// Use '0' for an unlimited number of revisions; defaults to '10'.
	// +optional
	MaxHistory *int `json:"maxHistory,omitempty"`

	// The name of the Kubernetes service account to impersonate
	// when reconciling this HelmRelease.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// PersistentClient tells the controller to use a persistent Kubernetes
	// client for this release. When enabled, the client will be reused for the
	// duration of the reconciliation, instead of being created and destroyed
	// for each (step of a) Helm action.
	//
	// This can improve performance, but may cause issues with some Helm charts
	// that for example do create Custom Resource Definitions during installation
	// outside Helm's CRD lifecycle hooks, which are then not observed to be
	// available by e.g. post-install hooks.
	//
	// If not set, it defaults to true.
	//
	// +optional
	PersistentClient *bool `json:"persistentClient,omitempty"`

	// Install holds the configuration for Helm install actions for this HelmRelease.
	// +optional
	Install *helmv2b1.Install `json:"install,omitempty"`

	// Upgrade holds the configuration for Helm upgrade actions for this HelmRelease.
	// +optional
	Upgrade *helmv2b1.Upgrade `json:"upgrade,omitempty"`

	// Rollback holds the configuration for Helm rollback actions for this HelmRelease.
	// +optional
	Rollback *helmv2b1.Rollback `json:"rollback,omitempty"`

	// Uninstall holds the configuration for Helm uninstall actions for this HelmRelease.
	// +optional
	Uninstall *helmv2b1.Uninstall `json:"uninstall,omitempty"`

	// ValuesFrom holds references to resources containing Helm values for this HelmRelease,
	// and information about how they should be merged.
	ValuesFrom []helmv2b1.ValuesReference `json:"valuesFrom,omitempty"`

	// Values holds the values for this Helm release.
	// +optional
	Values *apiextensionsv1.JSON `json:"values,omitempty"`
}

// HelmChartTemplate defines the template from which the controller will
// generate a v1beta2.HelmChart object in the same namespace as the referenced
// v1beta2.Source.
type HelmChartTemplate struct {
	// ObjectMeta holds the template for metadata like labels and annotations.
	// +optional
	ObjectMeta *HelmChartTemplateObjectMeta `json:"metadata,omitempty"`

	// Spec holds the template for the v1beta2.HelmChartSpec for this HelmRelease.
	// +required
	Spec HelmChartTemplateSpec `json:"spec"`
}

// HelmChartTemplateObjectMeta defines the template for the ObjectMeta of a
// v1beta2.HelmChart.
type HelmChartTemplateObjectMeta struct {
	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// HelmChartTemplateSpec defines the template from which the controller will
// generate a v1beta2.HelmChartSpec object.
type HelmChartTemplateSpec struct {
	// The name or path the Helm chart is available at in the SourceRef.
	// +required
	Chart string `json:"chart"`

	// Version semver expression, ignored for charts from v1beta2.GitRepository and
	// v1beta2.Bucket sources. Defaults to latest when omitted.
	// +kubebuilder:default:=*
	// +optional
	Version string `json:"version,omitempty"`

	// Interval at which to check the v1beta2.Source for updates. Defaults to
	// 'HelmReleaseSpec.Interval'.
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^([0-9]+(\\.[0-9]+)?(ms|s|m|h))+$"
	// +optional
	Interval *metav1.Duration `json:"interval,omitempty"`

	// Determines what enables the creation of a new artifact. Valid values are
	// ('ChartVersion', 'Revision').
	// See the documentation of the values for an explanation on their behavior.
	// Defaults to ChartVersion when omitted.
	// +kubebuilder:validation:Enum=ChartVersion;Revision
	// +kubebuilder:default:=ChartVersion
	// +optional
	ReconcileStrategy string `json:"reconcileStrategy,omitempty"`

	// Alternative list of values files to use as the chart values (values.yaml
	// is not included by default), expected to be a relative path in the SourceRef.
	// Values files are merged in the order of this list with the last file overriding
	// the first. Ignored when omitted.
	// +optional
	ValuesFiles []string `json:"valuesFiles,omitempty"`
}
