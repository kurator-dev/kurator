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

package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kurator.dev/kurator/pkg/apis/infra/v1alpha1"
)

func TestGetHostsContent(t *testing.T) {
	master1 := v1alpha1.Machine{
		HostName:  "master1",
		PrivateIP: "1.1.1.1",
		PublicIP:  "2.2.2.2",
	}
	node1 := v1alpha1.Machine{
		HostName:  "node1",
		PrivateIP: "3.3.3.3",
		PublicIP:  "4.4.4.4",
	}

	curCustomMachine := &v1alpha1.CustomMachine{
		Spec: v1alpha1.CustomMachineSpec{
			Master: []v1alpha1.Machine{master1},
			Nodes:  []v1alpha1.Machine{node1},
		},
	}

	expectHost := &HostTemplateContent{
		NodeAndIP:    []string{"master1 ansible_host=2.2.2.2 ip=1.1.1.1", "node1 ansible_host=4.4.4.4 ip=3.3.3.3"},
		MasterName:   []string{"master1"},
		NodeName:     []string{"node1"},
		EtcdNodeName: []string{"master1"},
	}
	assert.Equal(t, expectHost, GetHostsContent(curCustomMachine))
}
