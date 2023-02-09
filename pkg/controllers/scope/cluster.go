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
	"k8s.io/apimachinery/pkg/types"

	infrav1 "kurator.dev/kurator/pkg/apis/infra/v1alpha1"
)

type Cluster struct {
	types.NamespacedName
	Credential   string
	Version      string
	Region       string
	PodCIDR      []string
	ServiceCIDR  []string
	S3BucketName *string
	ControlPlane *Instance
	Workers      []*Instance
}

type Instance struct {
	Replicas     int
	InstanceType string
	SSHKey       string
	ImageOS      string
}

func NewCluster(cluster *infrav1.Cluster, credSecretName string) *Cluster {
	c := &Cluster{
		NamespacedName: types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		},
		Credential:   credSecretName,
		Version:      cluster.Spec.Version,
		Region:       cluster.Spec.Region,
		PodCIDR:      cluster.Spec.Network.PodCIDRs,
		ServiceCIDR:  cluster.Spec.Network.ServiceCIDRs,
		ControlPlane: NewInstance(cluster.Spec.Master.MachineConfig),
	}

	c.Workers = make([]*Instance, 0, len(cluster.Spec.Workers))
	for _, worker := range cluster.Spec.Workers {
		c.Workers = append(c.Workers, NewInstance(worker.MachineConfig))
	}

	return c
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
