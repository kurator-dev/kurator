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
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kurator.dev/kurator/e2e/resources"
	clusterv1a1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	fleetv1a1 "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
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
		namespace = "e2e-test"
		fleetname = "e2etest"
		memberClusterName = "kurator-member"
		homeDir, err := os.UserHomeDir()
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		kubeconfigPath = filepath.Join(homeDir, ".kube/kurator-member.config")

		// create namespace for e2e test
		e2eNamespace := resources.NewNamespace(namespace)
		createNSErr := resources.CreateNamespace(kubeClient, e2eNamespace)
		gomega.Expect(createNSErr).ShouldNot(gomega.HaveOccurred())
		time.Sleep(3 * time.Second)

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

	ginkgo.AfterEach(func() {
		fleerRemoveErr := resources.RemoveFleet(kuratorClient, namespace, fleetname)
		gomega.Expect(fleerRemoveErr).ShouldNot(gomega.HaveOccurred())

		attachedclusterRemoveErr := resources.RemoveAttachedCluster(kuratorClient, namespace, memberClusterName)
		gomega.Expect(attachedclusterRemoveErr).ShouldNot(gomega.HaveOccurred())

		secretRemoveErr := resources.RemoveSecret(kubeClient, namespace, memberClusterName)
		gomega.Expect(secretRemoveErr).ShouldNot(gomega.HaveOccurred())

		namespaceRemoveErr := resources.RemoveNamespace(kubeClient, namespace)
		gomega.Expect(namespaceRemoveErr).ShouldNot(gomega.HaveOccurred())
	})

	ginkgo.It("Create Fleet", func() {
		// step 1.create secrets
		secretCreateErr := resources.CreateSecret(kubeClient, secret)
		gomega.Expect(secretCreateErr).ShouldNot(gomega.HaveOccurred())

		// step 2.create attachedclusters
		attachedCreateErr := resources.CreateAttachedCluster(kuratorClient, attachedcluster)
		gomega.Expect(attachedCreateErr).ShouldNot(gomega.HaveOccurred())
		resources.WaitAttachedClusterFitWith(kuratorClient, namespace, memberClusterName, func(attachedCluster *clusterv1a1.AttachedCluster) bool {
			return attachedCluster.Status.Ready
		})

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
		time.Sleep(3 * time.Second)

		// step 4.check fleet status
		fleetPresentOnCluster, fleetGetErr := kuratorClient.FleetV1alpha1().Fleets(namespace).Get(context.TODO(), fleetname, metav1.GetOptions{})
		gomega.Expect(fleetGetErr).ShouldNot(gomega.HaveOccurred())
		gomega.Expect(fleetPresentOnCluster.Status.Phase).Should(gomega.Equal(fleetv1a1.ReadyPhase))
	})
})
