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

package scope

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	"kurator.dev/kurator/pkg/infra/util"
)

const (
	VpcDefaultCIDR = "10.0.0.0/16"

	ClusterNameLabel      = "cluster.kurator.dev/cluster-name"
	ClusterNamespaceLabel = "cluster.kurator.dev/cluster-namespace"
)

type Cluster struct {
	UID       string
	InfraType infrav1.ClusterInfraType
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
}

func NewCluster(cluster *infrav1.Cluster) *Cluster {
	nn := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}
	uid := util.GenerateUID(nn)
	c := &Cluster{
		InfraType:           cluster.Spec.InfraType,
		CNIType:             "calico",
		UID:                 uid,
		NamespacedName:      nn,
		Version:             cluster.Spec.Version,
		Region:              cluster.Spec.Region,
		VpcCIDR:             cluster.Spec.Network.VPC.CIDRBlock,
		PodCIDR:             cluster.Spec.Network.PodCIDRs,
		CredentialSecretRef: cluster.Spec.Credential.SecretRef,
		ServiceCIDR:         cluster.Spec.Network.ServiceCIDRs,
		ControlPlane:        NewInstance(cluster.Spec.Master.MachineConfig),
		EnablePodIdentity:   cluster.Spec.PodIdentity.Enabled,
	}

	if c.VpcCIDR == "" {
		c.VpcCIDR = VpcDefaultCIDR
	}

	if cluster.Spec.Network.CNI.Type != "" {
		c.CNIType = cluster.Spec.Network.CNI.Type
	}

	c.Workers = make([]*Instance, 0, len(cluster.Spec.Workers))
	for _, worker := range cluster.Spec.Workers {
		c.Workers = append(c.Workers, NewInstance(worker.MachineConfig))
	}

	c.BucketName = fmt.Sprintf("kuratorcluster-%s", c.UID)
	return c
}

func (c *Cluster) SecretName() string {
	return fmt.Sprintf("%s-%s-%s", c.Namespace, c.Name, c.UID)
}

func (c *Cluster) StackSuffix() string {
	return fmt.Sprintf(".%s-%s-%s.cluster.kurator.dev", c.Namespace, c.Name, c.UID)
}

func (c *Cluster) StackName() string {
	// statck must satisfy regular expression pattern: [a-zA-Z][-a-zA-Z0-9]*|arn:[-a-zA-Z0-9:/._+]*
	return fmt.Sprintf("%s-%s-%s-cluster-kurator-dev", c.Namespace, c.Name, c.UID)
}

func (c *Cluster) MatchingLabels() ctrlclient.MatchingLabels {
	return ctrlclient.MatchingLabels{
		ClusterNameLabel:      c.Name,
		ClusterNamespaceLabel: c.Namespace,
	}
}

func NewInstance(machine infrav1.MachineConfig) *Instance {
	inst := &Instance{
		Replicas:     machine.Replicas,
		InstanceType: machine.InstanceType,
		SSHKey:       machine.SSHKeyName,
		ImageOS:      machine.ImageOS,
	}

	return inst
}
