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
		namespace         string
		fleetname         string
		memberClusterName string
		kubeconfigPath    string
		secret            *corev1.Secret
		attachedcluster   *clusterv1a1.AttachedCluster
	)

	ginkgo.BeforeEach(func() {
		namespace = "default"
		fleetname = "e2etest"
		memberClusterName = "kurator-member"
		kubeconfigPath = "/root/.kube/kurator-member.config"

		// build secrets use member cluster kubeconfig
		kubeconfig, readfileErr := os.ReadFile(kubeconfigPath)
		gomega.Expect(readfileErr).ShouldNot(gomega.HaveOccurred())
		data := make(map[string][]byte)
		data[memberClusterName] = kubeconfig
		secret = resources.NewSecret(namespace, memberClusterName, data)

		// build two attachedclusters
		secretKeyRef := clusterv1a1.SecretKeyRef{
			Name: memberClusterName,
			Key:  memberClusterName,
		}
		attachedcluster = resources.NewAttachedCluster(namespace, memberClusterName, secretKeyRef)
	})

	ginkgo.It("Create Fleet", func() {
		// step 1.create secrets
		secretCreateErr := resources.CreateSecret(kubeClient, secret)
		gomega.Expect(secretCreateErr).ShouldNot(gomega.HaveOccurred())

		// step 2.create attachedclusters
		attachedCreateErr := resources.CreateAttachedCluster(kuratorClient, attachedcluster)
		gomega.Expect(attachedCreateErr).ShouldNot(gomega.HaveOccurred())

		time.Sleep(5 * time.Second)
		// step 3.create fleet
		clusters := []*corev1.ObjectReference{
			{
				Name: memberClusterName,
				Kind: "AttachedCluster",
			},
		}
		fleet := resources.NewFleet(namespace, fleetname, clusters)
		fleetCreateErr := resources.CreateFleet(kuratorClient, fleet)
		gomega.Expect(fleetCreateErr).ShouldNot(gomega.HaveOccurred())
	})
})
