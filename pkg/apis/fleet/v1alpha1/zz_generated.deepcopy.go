//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupConfig) DeepCopyInto(out *BackupConfig) {
	*out = *in
	if in.Chart != nil {
		in, out := &in.Chart, &out.Chart
		*out = new(ChartConfig)
		**out = **in
	}
	in.Storage.DeepCopyInto(&out.Storage)
	in.ExtraArgs.DeepCopyInto(&out.ExtraArgs)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupConfig.
func (in *BackupConfig) DeepCopy() *BackupConfig {
	if in == nil {
		return nil
	}
	out := new(BackupConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupStorage) DeepCopyInto(out *BackupStorage) {
	*out = *in
	in.Location.DeepCopyInto(&out.Location)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupStorage.
func (in *BackupStorage) DeepCopy() *BackupStorage {
	if in == nil {
		return nil
	}
	out := new(BackupStorage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupStorageLocation) DeepCopyInto(out *BackupStorageLocation) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupStorageLocation.
func (in *BackupStorageLocation) DeepCopy() *BackupStorageLocation {
	if in == nil {
		return nil
	}
	out := new(BackupStorageLocation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChartConfig) DeepCopyInto(out *ChartConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChartConfig.
func (in *ChartConfig) DeepCopy() *ChartConfig {
	if in == nil {
		return nil
	}
	out := new(ChartConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Config) DeepCopyInto(out *Config) {
	*out = *in
	if in.Chart != nil {
		in, out := &in.Chart, &out.Chart
		*out = new(ChartConfig)
		**out = **in
	}
	in.ExtraArgs.DeepCopyInto(&out.ExtraArgs)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Config.
func (in *Config) DeepCopy() *Config {
	if in == nil {
		return nil
	}
	out := new(Config)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Device) DeepCopyInto(out *Device) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Device.
func (in *Device) DeepCopy() *Device {
	if in == nil {
		return nil
	}
	out := new(Device)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DistributedStorage) DeepCopyInto(out *DistributedStorage) {
	*out = *in
	if in.DataDirHostPath != nil {
		in, out := &in.DataDirHostPath, &out.DataDirHostPath
		*out = new(string)
		**out = **in
	}
	if in.Monitor != nil {
		in, out := &in.Monitor, &out.Monitor
		*out = new(MonSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Manager != nil {
		in, out := &in.Manager, &out.Manager
		*out = new(MgrSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Storage != nil {
		in, out := &in.Storage, &out.Storage
		*out = new(StorageScopeSpec)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DistributedStorage.
func (in *DistributedStorage) DeepCopy() *DistributedStorage {
	if in == nil {
		return nil
	}
	out := new(DistributedStorage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DistributedStorageConfig) DeepCopyInto(out *DistributedStorageConfig) {
	*out = *in
	if in.Chart != nil {
		in, out := &in.Chart, &out.Chart
		*out = new(ChartConfig)
		**out = **in
	}
	if in.Storage != nil {
		in, out := &in.Storage, &out.Storage
		*out = new(DistributedStorage)
		(*in).DeepCopyInto(*out)
	}
	in.ExtraArgs.DeepCopyInto(&out.ExtraArgs)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DistributedStorageConfig.
func (in *DistributedStorageConfig) DeepCopy() *DistributedStorageConfig {
	if in == nil {
		return nil
	}
	out := new(DistributedStorageConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in Endpoints) DeepCopyInto(out *Endpoints) {
	{
		in := &in
		*out = make(Endpoints, len(*in))
		copy(*out, *in)
		return
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Endpoints.
func (in Endpoints) DeepCopy() Endpoints {
	if in == nil {
		return nil
	}
	out := new(Endpoints)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FlaggerConfig) DeepCopyInto(out *FlaggerConfig) {
	*out = *in
	if in.Chart != nil {
		in, out := &in.Chart, &out.Chart
		*out = new(ChartConfig)
		**out = **in
	}
	in.ExtraArgs.DeepCopyInto(&out.ExtraArgs)
	if in.ProviderConfig != nil {
		in, out := &in.ProviderConfig, &out.ProviderConfig
		*out = new(Config)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FlaggerConfig.
func (in *FlaggerConfig) DeepCopy() *FlaggerConfig {
	if in == nil {
		return nil
	}
	out := new(FlaggerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Fleet) DeepCopyInto(out *Fleet) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Fleet.
func (in *Fleet) DeepCopy() *Fleet {
	if in == nil {
		return nil
	}
	out := new(Fleet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Fleet) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FleetList) DeepCopyInto(out *FleetList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Fleet, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FleetList.
func (in *FleetList) DeepCopy() *FleetList {
	if in == nil {
		return nil
	}
	out := new(FleetList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *FleetList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FleetSpec) DeepCopyInto(out *FleetSpec) {
	*out = *in
	if in.Clusters != nil {
		in, out := &in.Clusters, &out.Clusters
		*out = make([]*v1.ObjectReference, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(v1.ObjectReference)
				**out = **in
			}
		}
	}
	if in.Plugin != nil {
		in, out := &in.Plugin, &out.Plugin
		*out = new(PluginConfig)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FleetSpec.
func (in *FleetSpec) DeepCopy() *FleetSpec {
	if in == nil {
		return nil
	}
	out := new(FleetSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FleetStatus) DeepCopyInto(out *FleetStatus) {
	*out = *in
	if in.CredentialSecret != nil {
		in, out := &in.CredentialSecret, &out.CredentialSecret
		*out = new(string)
		**out = **in
	}
	if in.PluginEndpoints != nil {
		in, out := &in.PluginEndpoints, &out.PluginEndpoints
		*out = make(map[string]Endpoints, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make(Endpoints, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FleetStatus.
func (in *FleetStatus) DeepCopy() *FleetStatus {
	if in == nil {
		return nil
	}
	out := new(FleetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaConfig) DeepCopyInto(out *GrafanaConfig) {
	*out = *in
	if in.Chart != nil {
		in, out := &in.Chart, &out.Chart
		*out = new(ChartConfig)
		**out = **in
	}
	in.ExtraArgs.DeepCopyInto(&out.ExtraArgs)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaConfig.
func (in *GrafanaConfig) DeepCopy() *GrafanaConfig {
	if in == nil {
		return nil
	}
	out := new(GrafanaConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KyvernoConfig) DeepCopyInto(out *KyvernoConfig) {
	*out = *in
	if in.Chart != nil {
		in, out := &in.Chart, &out.Chart
		*out = new(ChartConfig)
		**out = **in
	}
	if in.PodSecurity != nil {
		in, out := &in.PodSecurity, &out.PodSecurity
		*out = new(PodSecurityPolicy)
		**out = **in
	}
	in.ExtraArgs.DeepCopyInto(&out.ExtraArgs)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KyvernoConfig.
func (in *KyvernoConfig) DeepCopy() *KyvernoConfig {
	if in == nil {
		return nil
	}
	out := new(KyvernoConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricConfig) DeepCopyInto(out *MetricConfig) {
	*out = *in
	in.Thanos.DeepCopyInto(&out.Thanos)
	in.Prometheus.DeepCopyInto(&out.Prometheus)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricConfig.
func (in *MetricConfig) DeepCopy() *MetricConfig {
	if in == nil {
		return nil
	}
	out := new(MetricConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MgrSpec) DeepCopyInto(out *MgrSpec) {
	*out = *in
	if in.Count != nil {
		in, out := &in.Count, &out.Count
		*out = new(int)
		**out = **in
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Placement != nil {
		in, out := &in.Placement, &out.Placement
		*out = new(Placement)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MgrSpec.
func (in *MgrSpec) DeepCopy() *MgrSpec {
	if in == nil {
		return nil
	}
	out := new(MgrSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MonSpec) DeepCopyInto(out *MonSpec) {
	*out = *in
	if in.Count != nil {
		in, out := &in.Count, &out.Count
		*out = new(int)
		**out = **in
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Placement != nil {
		in, out := &in.Placement, &out.Placement
		*out = new(Placement)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MonSpec.
func (in *MonSpec) DeepCopy() *MonSpec {
	if in == nil {
		return nil
	}
	out := new(MonSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Node) DeepCopyInto(out *Node) {
	*out = *in
	in.StorageDeviceSelection.DeepCopyInto(&out.StorageDeviceSelection)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Node.
func (in *Node) DeepCopy() *Node {
	if in == nil {
		return nil
	}
	out := new(Node)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ObjectStoreConfig) DeepCopyInto(out *ObjectStoreConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ObjectStoreConfig.
func (in *ObjectStoreConfig) DeepCopy() *ObjectStoreConfig {
	if in == nil {
		return nil
	}
	out := new(ObjectStoreConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Placement) DeepCopyInto(out *Placement) {
	*out = *in
	if in.NodeAffinity != nil {
		in, out := &in.NodeAffinity, &out.NodeAffinity
		*out = new(v1.NodeAffinity)
		(*in).DeepCopyInto(*out)
	}
	if in.PodAffinity != nil {
		in, out := &in.PodAffinity, &out.PodAffinity
		*out = new(v1.PodAffinity)
		(*in).DeepCopyInto(*out)
	}
	if in.PodAntiAffinity != nil {
		in, out := &in.PodAntiAffinity, &out.PodAntiAffinity
		*out = new(v1.PodAntiAffinity)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.TopologySpreadConstraints != nil {
		in, out := &in.TopologySpreadConstraints, &out.TopologySpreadConstraints
		*out = make([]v1.TopologySpreadConstraint, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Placement.
func (in *Placement) DeepCopy() *Placement {
	if in == nil {
		return nil
	}
	out := new(Placement)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PluginConfig) DeepCopyInto(out *PluginConfig) {
	*out = *in
	if in.Metric != nil {
		in, out := &in.Metric, &out.Metric
		*out = new(MetricConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Grafana != nil {
		in, out := &in.Grafana, &out.Grafana
		*out = new(GrafanaConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Policy != nil {
		in, out := &in.Policy, &out.Policy
		*out = new(PolicyConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Backup != nil {
		in, out := &in.Backup, &out.Backup
		*out = new(BackupConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.DistributedStorage != nil {
		in, out := &in.DistributedStorage, &out.DistributedStorage
		*out = new(DistributedStorageConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Flagger != nil {
		in, out := &in.Flagger, &out.Flagger
		*out = new(FlaggerConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.SubMarinerOperator != nil {
		in, out := &in.SubMarinerOperator, &out.SubMarinerOperator
		*out = new(SubMarinerOperatorConfig)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PluginConfig.
func (in *PluginConfig) DeepCopy() *PluginConfig {
	if in == nil {
		return nil
	}
	out := new(PluginConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodSecurityPolicy) DeepCopyInto(out *PodSecurityPolicy) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodSecurityPolicy.
func (in *PodSecurityPolicy) DeepCopy() *PodSecurityPolicy {
	if in == nil {
		return nil
	}
	out := new(PodSecurityPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PolicyConfig) DeepCopyInto(out *PolicyConfig) {
	*out = *in
	if in.Kyverno != nil {
		in, out := &in.Kyverno, &out.Kyverno
		*out = new(KyvernoConfig)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PolicyConfig.
func (in *PolicyConfig) DeepCopy() *PolicyConfig {
	if in == nil {
		return nil
	}
	out := new(PolicyConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrometheusConfig) DeepCopyInto(out *PrometheusConfig) {
	*out = *in
	if in.Chart != nil {
		in, out := &in.Chart, &out.Chart
		*out = new(ChartConfig)
		**out = **in
	}
	if in.NodeExporter != nil {
		in, out := &in.NodeExporter, &out.NodeExporter
		*out = new(PrometheusExporterConfig)
		**out = **in
	}
	if in.KubeStateMetrics != nil {
		in, out := &in.KubeStateMetrics, &out.KubeStateMetrics
		*out = new(PrometheusExporterConfig)
		**out = **in
	}
	in.ExtraArgs.DeepCopyInto(&out.ExtraArgs)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrometheusConfig.
func (in *PrometheusConfig) DeepCopy() *PrometheusConfig {
	if in == nil {
		return nil
	}
	out := new(PrometheusConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrometheusExporterConfig) DeepCopyInto(out *PrometheusExporterConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrometheusExporterConfig.
func (in *PrometheusExporterConfig) DeepCopy() *PrometheusExporterConfig {
	if in == nil {
		return nil
	}
	out := new(PrometheusExporterConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StorageDeviceSelection) DeepCopyInto(out *StorageDeviceSelection) {
	*out = *in
	if in.Devices != nil {
		in, out := &in.Devices, &out.Devices
		*out = make([]Device, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StorageDeviceSelection.
func (in *StorageDeviceSelection) DeepCopy() *StorageDeviceSelection {
	if in == nil {
		return nil
	}
	out := new(StorageDeviceSelection)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StorageScopeSpec) DeepCopyInto(out *StorageScopeSpec) {
	*out = *in
	if in.Nodes != nil {
		in, out := &in.Nodes, &out.Nodes
		*out = make([]Node, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.StorageDeviceSelection.DeepCopyInto(&out.StorageDeviceSelection)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StorageScopeSpec.
func (in *StorageScopeSpec) DeepCopy() *StorageScopeSpec {
	if in == nil {
		return nil
	}
	out := new(StorageScopeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SubMarinerOperatorConfig) DeepCopyInto(out *SubMarinerOperatorConfig) {
	*out = *in
	if in.Chart != nil {
		in, out := &in.Chart, &out.Chart
		*out = new(ChartConfig)
		**out = **in
	}
	in.ExtraArgs.DeepCopyInto(&out.ExtraArgs)
	if in.ClusterCidrs != nil {
		in, out := &in.ClusterCidrs, &out.ClusterCidrs
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ServiceCidrs != nil {
		in, out := &in.ServiceCidrs, &out.ServiceCidrs
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Globalcidrs != nil {
		in, out := &in.Globalcidrs, &out.Globalcidrs
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SubMarinerOperatorConfig.
func (in *SubMarinerOperatorConfig) DeepCopy() *SubMarinerOperatorConfig {
	if in == nil {
		return nil
	}
	out := new(SubMarinerOperatorConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ThanosConfig) DeepCopyInto(out *ThanosConfig) {
	*out = *in
	if in.Chart != nil {
		in, out := &in.Chart, &out.Chart
		*out = new(ChartConfig)
		**out = **in
	}
	out.ObjectStoreConfig = in.ObjectStoreConfig
	in.ExtraArgs.DeepCopyInto(&out.ExtraArgs)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ThanosConfig.
func (in *ThanosConfig) DeepCopy() *ThanosConfig {
	if in == nil {
		return nil
	}
	out := new(ThanosConfig)
	in.DeepCopyInto(out)
	return out
}
