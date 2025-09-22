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

package scope

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	"kurator.dev/kurator/pkg/infra/util"
)

const (
	VpcDefaultCIDR = "10.0.0.0/16"

	ClusterNameLabel      = "cluster.kurator.dev/cluster-name"
	ClusterNamespaceLabel = "cluster.kurator.dev/cluster-namespace"
	BucketNamePrefix      = "kuratorcluster-"
)

type Cluster struct {
	// record original object so we can mutate it directly
	Cluster   *clusterv1alpha1.Cluster
	UID       string
	InfraType clusterv1alpha1.ClusterInfraType
	types.NamespacedName
	CredentialSecretRef string
	Version             string
	Region              string
	VpcCIDR             string
	PodCIDR             []string
	ServiceCIDR         []string
	CNIType             string

	ControlPlane *Instance
	Workers      []*Instance

	EnablePodIdentity bool
	BucketName        string
}

type Instance struct {
	Replicas     int
	InstanceType string
	SSHKey       string
	ImageOS      string

	RootVolume  *InstanceVolume
	DataVolumes []InstanceVolume
}

type InstanceVolume struct {
	DeviceName string
	Size       int64
	Type       string
}

func NewCluster(cluster *clusterv1alpha1.Cluster) *Cluster {
	nn := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}
	uid := util.GenerateUID(nn)
	c := &Cluster{
		Cluster:           cluster,
		InfraType:         cluster.Spec.InfraType,
		CNIType:           "calico",
		UID:               uid,
		NamespacedName:    nn,
		Version:           cluster.Spec.Version,
		Region:            cluster.Spec.Region,
		VpcCIDR:           cluster.Spec.Network.VPC.CIDRBlock,
		PodCIDR:           cluster.Spec.Network.PodCIDRs,
		ServiceCIDR:       cluster.Spec.Network.ServiceCIDRs,
		ControlPlane:      NewInstance(cluster.Spec.InfraType, cluster.Spec.Master.MachineConfig),
		EnablePodIdentity: cluster.Spec.PodIdentity.Enabled,
	}

	if cluster.Spec.Credential != nil {
		c.CredentialSecretRef = cluster.Spec.Credential.SecretRef
	}

	if c.VpcCIDR == "" {
		c.VpcCIDR = VpcDefaultCIDR
	}

	if cluster.Spec.Network.CNI.Type != "" {
		c.CNIType = cluster.Spec.Network.CNI.Type
	}

	c.Workers = make([]*Instance, 0, len(cluster.Spec.Workers))
	for _, worker := range cluster.Spec.Workers {
		c.Workers = append(c.Workers, NewInstance(cluster.Spec.InfraType, worker.MachineConfig))
	}

	c.BucketName = fmt.Sprintf("%s%s", BucketNamePrefix, c.UID)
	return c
}

func (c *Cluster) SecretName() string {
	return fmt.Sprintf("%s-%s-%s", c.Namespace, c.Name, c.UID)
}

func (c *Cluster) StackSuffix() string {
	return fmt.Sprintf(".%s.kurator.dev", c.UID)
}

func (c *Cluster) StackName() string {
	// statck must satisfy regular expression pattern: [a-zA-Z][-a-zA-Z0-9]*|arn:[-a-zA-Z0-9:/._+]*
	return fmt.Sprintf("%s-%s-cluster-kurator-dev", "cf", c.UID)
}

func (c *Cluster) MatchingLabels() ctrlclient.MatchingLabels {
	return ctrlclient.MatchingLabels{
		ClusterNameLabel:      c.Name,
		ClusterNamespaceLabel: c.Namespace,
	}
}

func NewInstance(infraType clusterv1alpha1.ClusterInfraType, machine clusterv1alpha1.MachineConfig) *Instance {
	inst := &Instance{
		Replicas:     machine.Replicas,
		InstanceType: machine.InstanceType,
		SSHKey:       machine.SSHKeyName,
		ImageOS:      machine.ImageOS,
	}

	if machine.RootVolume != nil {
		inst.RootVolume = &InstanceVolume{
			Size: machine.RootVolume.Size,
			Type: machine.RootVolume.Type,
		}
	}

	if len(machine.NonRootVolumes) > 0 {
		inst.DataVolumes = make([]InstanceVolume, 0, len(machine.NonRootVolumes))
		for idx, vol := range machine.NonRootVolumes {
			inst.DataVolumes = append(inst.DataVolumes, InstanceVolume{
				Size:       vol.Size,
				Type:       vol.Type,
				DeviceName: deviceName(infraType, idx),
			})
		}
	}

	return inst
}

func deviceName(infra clusterv1alpha1.ClusterInfraType, idx int) string {
	if infra == clusterv1alpha1.AWSClusterInfraType {
		// for AWS, device name is /dev/sd[b-z]
		return fmt.Sprintf("/dev/sd%c", 'b'+idx)
	}

	return "Not implemented"
}
