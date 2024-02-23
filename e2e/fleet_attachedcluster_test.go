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

package e2e

import (
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"kurator.dev/kurator/e2e/resources"
	fleetv1a1 "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

var _ = ginkgo.Describe("[AttachedClusters] AttachedClusters testing", func() {
	var (
		fleetNamespace string
		fleetname      string
		fleet          *fleetv1a1.Fleet
	)

	ginkgo.BeforeEach(func() {
		fleetNamespace = e2ePrefix + resources.RandomNamespace(4)
		fleetname = "e2e"
		// build fleet
		clusters := []*corev1.ObjectReference{
			{
				Name: memberClusterName,
				Kind: "AttachedCluster",
			},
		}
		fleet = resources.NewFleet(fleetNamespace, fleetname, clusters)
	})

	ginkgo.AfterEach(func() {
		fleerRemoveErr := resources.RemoveFleet(kuratorClient, fleetNamespace, fleetname)
		gomega.Expect(fleerRemoveErr).ShouldNot(gomega.HaveOccurred())

		attachedclusterRemoveErr := resources.RemoveAttachedCluster(kuratorClient, namespace, memberClusterName)
		gomega.Expect(attachedclusterRemoveErr).ShouldNot(gomega.HaveOccurred())

		secretRemoveErr := resources.RemoveSecret(kubeClient, namespace, memberClusterName)
		gomega.Expect(secretRemoveErr).ShouldNot(gomega.HaveOccurred())

		namespaceRemoveErr := resources.RemoveNamespace(kubeClient, fleetNamespace)
		gomega.Expect(namespaceRemoveErr).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("Create Fleet", func() {
		// create a namespace for fleet e2e test
		fleetNamespaceCfg := resources.NewNamespace(fleetNamespace)
		createNSErr := resources.CreateNamespace(kubeClient, fleetNamespaceCfg)
		gomega.Expect(createNSErr).ShouldNot(gomega.HaveOccurred())
		time.Sleep(3 * time.Second)
		// create fleet and checkout fleet status
		fleetCreateErr := resources.CreateFleet(kuratorClient, fleet)
		gomega.Expect(fleetCreateErr).ShouldNot(gomega.HaveOccurred())
		resources.WaitFleetReady(kuratorClient, fleetNamespace, fleetname, func(fleet *fleetv1a1.Fleet) bool {
			return fleet.Status.Phase == fleetv1a1.ReadyPhase
		})
	})
})
