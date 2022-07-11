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

package submariner

import (
	"context"
	"fmt"
	"os/exec"
	"path"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"kurator.dev/kurator/pkg/util"
)

func (p *SubmarinerPlugin) runInstall() error {
	// 1.generate kubeconfig for each cluster
	memClusters, err := p.generateKubeConfiguration()
	if err != nil {
		return err
	}
	if len(memClusters) < 2 {
		return fmt.Errorf("memcluster number %d < 2", len(memClusters))
	}
	// 2. install broker in the first memcluster
	err = p.installBroker(memClusters[0])
	if err != nil {
		return fmt.Errorf("install broker failed: %v", err)
	}
	// 3. connect all clusters to the Broker
	err = p.connectBroker(memClusters)
	if err != nil {
		return fmt.Errorf("connect broker failed: %v", err)
	}
	return nil
}

func (p *SubmarinerPlugin) installBroker(kubeconfig string) error {
	// subctl deploy-broker --kubeconfig /root/.kube/karmada.config --kubecontext karmada-host
	installArgs := []string{
		"deploy-broker",
		"--kubeconfig",
		kubeconfig,
	}

	logrus.Debugf("run cmd: %s %v", p.subctl, installArgs)
	cmd := exec.Command(p.subctl, installArgs...)
	err := util.RunCommand(cmd)
	return err
}

func (p *SubmarinerPlugin) connectBroker(kubeconfigs []string) error {
	// subctl join --kubeconfig /root/.kube/cluster1.config broker-info.subm --natt=false
	installArgs := []string{
		"join",
		"broker-info.subm",
		"--natt=false",
	}
	for _, kubeconfig := range kubeconfigs {
		args := append(installArgs, "--kubeconfig", kubeconfig)
		logrus.Debugf("run cmd: %s %v", p.subctl, args)
		cmd := exec.Command(p.subctl, args...)
		if err := util.RunCommand(cmd); err != nil {
			return err
		}
	}
	return nil
}

func (p *SubmarinerPlugin) generateKubeConfiguration() ([]string, error) {
	clusterList, err := p.KarmadaClient().ClusterV1alpha1().Clusters().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list karmada clusters fail, %w", err)
	}

	var out []string
	// Install dir  filepath.Join(p.options.HomeDir, istioComponent.Name, istioComponent.Version)
	for _, cluster := range clusterList.Items {
		clusters := make(map[string]*clientcmdapi.Cluster)
		secretMeta := cluster.Spec.SecretRef
		secret, err := p.KubeClient().CoreV1().Secrets(secretMeta.Namespace).Get(context.TODO(), secretMeta.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		clusters["default-cluster"] = &clientcmdapi.Cluster{
			Server:                   cluster.Spec.APIEndpoint,
			CertificateAuthorityData: secret.Data["caBundle"],
		}

		contexts := make(map[string]*clientcmdapi.Context)
		contexts["default-context"] = &clientcmdapi.Context{
			Cluster:  "default-cluster",
			AuthInfo: "default-context",
		}

		authinfos := make(map[string]*clientcmdapi.AuthInfo)
		authinfos["default-context"] = &clientcmdapi.AuthInfo{
			Token: string(secret.Data["token"]),
		}

		clientConfig := clientcmdapi.Config{
			Kind:           "Config",
			APIVersion:     "v1",
			Clusters:       clusters,
			Contexts:       contexts,
			CurrentContext: "default-context",
			AuthInfos:      authinfos,
		}
		kubeconfig := path.Join(p.installPath, cluster.Name+".kubeconfig")
		err = clientcmd.WriteToFile(clientConfig, kubeconfig)
		if err != nil {
			return nil, err
		}
		out = append(out, kubeconfig)
	}
	return out, nil
}
