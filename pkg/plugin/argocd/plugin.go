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

package argocd

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"k8s.io/apimachinery/pkg/types"

	karmadautil "github.com/karmada-io/karmada/pkg/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sirupsen/logrus"
	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/moreos"
	"kurator.dev/kurator/pkg/util"
)

var argocd = filepath.Join("argocd" + moreos.Exe)

type InstallArgs struct {
	// basically specify the karmada apiserver kubeconfig, where argocd deploy apps to.
	ClusterKubeconfig string
	ClusterContext    string
}

type ArgoCDPlugin struct {
	*client.Client

	options     *generic.Options
	args        *InstallArgs
	argocdCli   string
	installPath string
}

func NewArgoCDPlugin(s *generic.Options, args *InstallArgs) (*ArgoCDPlugin, error) {
	plugin := &ArgoCDPlugin{
		options:   s,
		args:      args,
		argocdCli: "/usr/local/bin/argocd",
	}
	rest := s.RESTClientGetter()
	c, err := client.NewClient(rest)
	if err != nil {
		return nil, err
	}
	plugin.Client = c

	return plugin, nil
}

func (p *ArgoCDPlugin) init() error {
	cli, err := p.installCli()
	if err != nil {
		return err
	}
	p.argocdCli = cli
	return nil
}

// Execute receives an executable's filepath, a slice
// of arguments, and a slice of environment variables
// to relay to the executable.
func (p *ArgoCDPlugin) Execute(cmdArgs, environment []string) error {
	if err := p.init(); err != nil {
		return err
	}

	if err := p.installArgoCD(); err != nil {
		logrus.Infof("failed to install argocd, %s", err)
		return err
	}

	// Add cluster karmada-apiserver to deploy apps to.
	if err := p.addCluster(); err != nil {
		logrus.Infof("failed to add cluster: %s", err)
		return err
	}

	return nil
}

func (p *ArgoCDPlugin) installCli() (string, error) {
	istioComponent := p.options.Components["argocd"]
	p.installPath = filepath.Join(p.options.HomeDir, istioComponent.Name, istioComponent.Version)
	argocdBinaryPath := filepath.Join(p.installPath, "argocd")
	_, err := os.Stat(argocdBinaryPath)
	if err == nil {
		return argocdBinaryPath, nil
	}

	if os.IsNotExist(err) {
		if err = os.MkdirAll(p.installPath, 0o750); err != nil {
			return "", fmt.Errorf("unable to create directory %q: %w", p.installPath, err)
		}
		// https://github.com/argoproj/argo-cd/releases/download/v2.4.8/argocd-linux-amd64
		argocdCli := fmt.Sprintf("argocd-%s-%s", util.OSExt(), runtime.GOARCH)
		url, _ := util.JoinUrlPath(istioComponent.ReleaseURLPrefix, istioComponent.Version, argocdCli)
		// download argocd
		if _, err := util.DownloadResource(url, argocdBinaryPath); err != nil {
			return "", fmt.Errorf("unable to get argocd cli %q: %w", argocdBinaryPath, err)
		}
	}

	// chmod +x
	os.Chmod(argocdBinaryPath, 0777)
	return util.VerifyExecutableBinary(argocdBinaryPath)
}

func (p *ArgoCDPlugin) argocdManifest() (string, error) {
	// TODO: refactor component to support both cli and manifest url
	// https://raw.githubusercontent.com/argoproj/argo-cd/v2.4.8/manifests/install.yaml
	url := "https://raw.githubusercontent.com/argoproj/argo-cd/v2.4.8/manifests/install.yaml"
	yaml, err := util.DownloadResource(url, "")
	if err != nil {
		return "", err
	}

	return yaml, nil
}

func (p *ArgoCDPlugin) installArgoCD() error {
	if _, err := karmadautil.EnsureNamespaceExist(p.KubeClient(), "argocd", false); err != nil {
		return fmt.Errorf("failed to ersure namespace %s, %w", "argocd", err)
	}

	argocdYaml, err := p.argocdManifest()
	if err != nil {
		return err
	}

	helmClient := p.HelmClient()
	helmClient.Namespace = "argocd"
	resourceList, err := helmClient.Build(bytes.NewBufferString(argocdYaml), false)
	if err != nil {
		return err
	}

	if p.options.DryRun {
		logrus.Infof("apply resoucrs: %s", argocdYaml)
		return nil
	}

	if _, err := helmClient.Create(resourceList); err != nil {
		return err
	}

	// kubectl patch svc argocd-server -n argocd -p '{"spec": {"type": "LoadBalancer"}}'
	patch := fmt.Sprintf(`[{"op": "replace", "path": "/spec/type", "value":"LoadBalancer"}]`)
	_, err = p.KubeClient().CoreV1().Services("argocd").Patch(context.TODO(), "argocd-server", types.JSONPatchType,
		[]byte(patch), metav1.PatchOptions{})
	return err
}

// addCluster add karmada apiserver to deploy apps to
func (p *ArgoCDPlugin) addCluster() error {

	// TODO: wait
	// 1. get argocd server address lb ip
	svc, err := p.KubeClient().CoreV1().Services("argocd").Get(context.TODO(), "argocd-server", metav1.GetOptions{})
	if err != nil {
		return err
	}

	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		return fmt.Errorf("loadbalancer service is pending")
	}

	serverAddress := net.JoinHostPort(svc.Status.LoadBalancer.Ingress[0].IP, strconv.Itoa(int(svc.Spec.Ports[0].Port)))

	// 2. get password
	// kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d; echo
	secret, err := p.KubeClient().CoreV1().Secrets("argocd").Get(context.TODO(), "argocd-initial-admin-secret", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get passwd secret argocd-initial-admin-secret err: %v", err)
	}

	passwd, ok := secret.Data["password"]
	if !ok {
		return fmt.Errorf("passwd secret argocd-initial-admin-secret does not contain `passwd`")
	}

	// 3. login argocd login 127.0.0.1:30080 --password=bwSS-KlnBB1IlRZF --username=admin --insecure
	args := []string{
		"login",
		serverAddress,
		"--user=admin",
		fmt.Sprintf("--password=%s", passwd),
		"--insecure",
	}

	cmd := exec.Command(p.argocdCli, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Infof("%s", string(out))
		return err
	}

	// kubectl config get-contexts -o name
	// 4. argocd cluster add karmada --kubeconfig=/etc/karmada/karmada-apiserver.config -y
	args = []string{
		"cluster",
		"add",
		p.args.ClusterContext,
		fmt.Sprintf("--kubeconfig=%s", p.args.ClusterKubeconfig),
		"-y",
	}
	cmd = exec.Command(p.argocdCli, args...)
	out, err = cmd.CombinedOutput()
	if err != nil {
		logrus.Infof("%s", string(out))
		return err
	}

	return nil
}
