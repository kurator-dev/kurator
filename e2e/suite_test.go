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
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"kurator.dev/kurator/e2e/framework"
	"kurator.dev/kurator/e2e/resources"
	clusterv1a1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	kurator "kurator.dev/kurator/pkg/client-go/generated/clientset/versioned"
)

var (
	kubeconfig     string
	kubeClient     kubernetes.Interface
	kuratorClient  kurator.Interface
	kuratorContext string

	namespace         string
	memberClusterName string
	kubeconfigPath    string
	secret            *corev1.Secret
	attachedcluster   *clusterv1a1.AttachedCluster
)

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "E2E Suite")
}

var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
	return nil
}, func(bytes []byte) {
	kubeconfig = os.Getenv("KUBECONFIG")
	gomega.Expect(kubeconfig).ShouldNot(gomega.BeEmpty())

	rest, err := framework.LoadRESTClientConfig(kubeconfig, kuratorContext)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	kubeClient, err = kubernetes.NewForConfig(rest)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	kuratorClient, err = kurator.NewForConfig(rest)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	namespace = "e2e-test"
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

	// build attachedclusters for fleet
	secretKeyRef := clusterv1a1.SecretKeyRef{
		Name: memberClusterName,
		Key:  memberClusterName,
	}
	attachedcluster = resources.NewAttachedCluster(namespace, memberClusterName, secretKeyRef)

	secretCreateErr := resources.CreateSecret(kubeClient, secret)
	gomega.Expect(secretCreateErr).ShouldNot(gomega.HaveOccurred())

	attachedCreateErr := resources.CreateAttachedCluster(kuratorClient, attachedcluster)
	gomega.Expect(attachedCreateErr).ShouldNot(gomega.HaveOccurred())
	resources.WaitAttachedClusterFitWith(kuratorClient, namespace, memberClusterName, func(attachedCluster *clusterv1a1.AttachedCluster) bool {
		return attachedCluster.Status.Ready
	})
})
