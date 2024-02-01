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
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"

	"kurator.dev/kurator/e2e/framework"
	kurator "kurator.dev/kurator/pkg/client-go/generated/clientset/versioned"
)

var (
	kubeconfig     string
	kubeClient     kubernetes.Interface
	kuratorClient  kurator.Interface
	kuratorContext string
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
})
