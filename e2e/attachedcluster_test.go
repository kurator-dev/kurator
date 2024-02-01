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
	"os"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"kurator.dev/kurator/e2e/resources"
	clusterv1a1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
)

var _ = ginkgo.Describe("[AttachedClusters] AttachedClusters testing", func() {
	var (
		namespace          string
		fleetname          string
		memberClusterName1 string
		memberClusterName2 string
		kubeconfig1Path    string
		kubeconfig2Path    string
		secret1            *corev1.Secret
		secret2            *corev1.Secret
		attachedcluster1   *clusterv1a1.AttachedCluster
		attachedcluster2   *clusterv1a1.AttachedCluster
	)

	ginkgo.BeforeEach(func() {
		namespace = "default"
		fleetname = "e2etest"
		memberClusterName1 = "kurator-member1"
		memberClusterName2 = "kurator-member2"
		kubeconfig1Path = "/root/.kube/kurator-member1.config"
		kubeconfig2Path = "/root/.kube/kurator-member2.config"

		// build two secrets
		kubeconfig1, readfileErr1 := os.ReadFile(kubeconfig1Path)
		gomega.Expect(readfileErr1).ShouldNot(gomega.HaveOccurred())
		data1 := make(map[string][]byte)
		data1[memberClusterName1] = kubeconfig1
		secret1 = resources.NewSecret(namespace, memberClusterName1, data1)

		kubeconfig2, readfileErr2 := os.ReadFile(kubeconfig2Path)
		gomega.Expect(readfileErr2).ShouldNot(gomega.HaveOccurred())
		data2 := make(map[string][]byte)
		data2[memberClusterName2] = kubeconfig2
		secret2 = resources.NewSecret(namespace, memberClusterName2, data2)

		// build two attachedclusters
		secretKeyRef1 := clusterv1a1.SecretKeyRef{
			Name: memberClusterName1,
			Key:  memberClusterName1,
		}
		secretKeyRef2 := clusterv1a1.SecretKeyRef{
			Name: memberClusterName2,
			Key:  memberClusterName2,
		}
		attachedcluster1 = resources.NewAttachedCluster(namespace, memberClusterName1, secretKeyRef1)
		attachedcluster2 = resources.NewAttachedCluster(namespace, memberClusterName2, secretKeyRef2)
	})

	ginkgo.It("Create Fleet", func() {
		// step 1.create secrets
		secretCreateErr1 := resources.CreateSecret(kubeClient, secret1)
		secretCreateErr2 := resources.CreateSecret(kubeClient, secret2)
		gomega.Expect(secretCreateErr1).ShouldNot(gomega.HaveOccurred())
		gomega.Expect(secretCreateErr2).ShouldNot(gomega.HaveOccurred())

		// step 2.create attachedclusters
		attachedCreateErr1 := resources.CreateAttachedCluster(kuratorClient, attachedcluster1)
		attachedCreateErr2 := resources.CreateAttachedCluster(kuratorClient, attachedcluster2)
		gomega.Expect(attachedCreateErr1).ShouldNot(gomega.HaveOccurred())
		gomega.Expect(attachedCreateErr2).ShouldNot(gomega.HaveOccurred())

		time.Sleep(5 * time.Second)
		// step 3.create fleet
		clusters := []*corev1.ObjectReference{
			{
				Name: memberClusterName1,
				Kind: "AttachedCluster",
			},
			{
				Name: memberClusterName2,
				Kind: "AttachedCluster",
			},
		}
		fleet := resources.NewFleet(namespace, fleetname, clusters)
		fleetCreateErr := resources.CreateFleet(kuratorClient, fleet)
		gomega.Expect(fleetCreateErr).ShouldNot(gomega.HaveOccurred())
	})
})
