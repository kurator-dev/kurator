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

	karmadautil "github.com/karmada-io/karmada/pkg/util"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/util"
)

const (
	component     = "argocd"
	namespace     = "argocd"
	argocdService = "argocd-server"
	passwdSecret  = "argocd-initial-admin-secret"
)

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
		logrus.Errorf("failed to install argocd, %s", err)
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
	istioComponent := p.options.Components[component]
	p.installPath = filepath.Join(p.options.HomeDir, istioComponent.Name, istioComponent.Version)
	argocdBinaryPath := filepath.Join(p.installPath, component)
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
	if err := os.Chmod(argocdBinaryPath, 0750); err != nil {
		return "", err
	}
	return util.VerifyExecutableBinary(argocdBinaryPath)
}

func (p *ArgoCDPlugin) argocdManifest() (string, error) {
	// TODO: refactor component to support both cli and manifest url
	// https://raw.githubusercontent.com/argoproj/argo-cd/v2.4.8/manifests/install.yaml
	istioComponent := p.options.Components[component]
	url := fmt.Sprintf("https://raw.githubusercontent.com/argoproj/argo-cd/%s/manifests/install.yaml", istioComponent.Version)
	yaml, err := util.DownloadResource(url, "")
	if err != nil {
		return "", err
	}

	return yaml, nil
}

func (p *ArgoCDPlugin) installArgoCD() error {
	// Delete namespace in order to delete previous intermediate output resource created.
	// Like admin secret, which will influence idempotent.
	policy := metav1.DeletePropagationForeground
	if err := p.KubeClient().CoreV1().Namespaces().Delete(context.TODO(), namespace,
		metav1.DeleteOptions{PropagationPolicy: &policy}); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete namespace %s, %w", namespace, err)
		}
	}

	if _, err := karmadautil.EnsureNamespaceExist(p.KubeClient(), namespace, false); err != nil {
		return fmt.Errorf("failed to ensure namespace %s, %w", namespace, err)
	}

	argocdYaml, err := p.argocdManifest()
	if err != nil {
		return err
	}

	helmClient := p.HelmClient()
	helmClient.Namespace = namespace
	resourceList, err := helmClient.Build(bytes.NewBufferString(argocdYaml), false)
	if err != nil {
		return err
	}

	if p.options.DryRun {
		logrus.Infof("apply resoucrs: %s", argocdYaml)
		return nil
	}

	logrus.Debugf("apply argocd manifests")
	// first delete existing resources to make install idempotent
	if _, err := helmClient.Update(resourceList, resourceList, true); err != nil {
		return err
	}

	// kubectl patch svc argocd-server -n argocd -p '{"spec": {"type": "LoadBalancer"}}'
	patch := `[{"op": "replace", "path": "/spec/type", "value":"LoadBalancer"}]`
	if _, err = p.KubeClient().CoreV1().Services(namespace).Patch(context.TODO(), argocdService, types.JSONPatchType,
		[]byte(patch), metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("patching service %s: %v", argocdService, err)
	}

	logrus.Debugf("wait for resources ready")
	return helmClient.Wait(resourceList, p.options.WaitTimeout)
}

// addCluster add karmada apiserver to deploy apps to
func (p *ArgoCDPlugin) addCluster() error {
	// 1. get argocd server address lb ip
	svc, err := util.WaitServiceReady(p.KubeClient(), namespace, argocdService, p.options.WaitInterval, p.options.WaitTimeout)
	if err != nil {
		logrus.Debugf("argocd-server service not ready: %s", err)
		return err
	}
	serverAddress := net.JoinHostPort(svc.Status.LoadBalancer.Ingress[0].IP, strconv.Itoa(int(svc.Spec.Ports[0].Port)))
	logrus.Debugf("argocd-server address: %s", serverAddress)

	// 2. get password
	// kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d; echo
	secret, err := p.KubeClient().CoreV1().Secrets(namespace).Get(context.TODO(), passwdSecret, metav1.GetOptions{})
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
		"--username=admin",
		fmt.Sprintf("--password=%s", passwd),
		"--insecure",
	}

	logrus.Debugf("%s %v", p.argocdCli, args)
	cmd := exec.Command(p.argocdCli, args...)
	if err := util.RunCommand(cmd); err != nil {
		logrus.Errorf("%s %v: %s", p.argocdCli, args, err)
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
	logrus.Debugf("%s %v", p.argocdCli, args)
	cmd = exec.Command(p.argocdCli, args...)
	if err := util.RunCommand(cmd); err != nil {
		logrus.Errorf("%s %v: %s", p.argocdCli, args, err)
		return err
	}

	return nil
}
