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

package vizier

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	karmadautil "github.com/karmada-io/karmada/pkg/util"
	"github.com/sirupsen/logrus"
	helmclient "helm.sh/helm/v3/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"

	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/moreos"
	"kurator.dev/kurator/pkg/util"
)

const (
	crdKind       = "CustomResourceDefinition"
	vizierCRDName = "viziers.px.dev"

	repoName    = "pixie-operator"
	repoAddress = "https://pixie-operator-charts.storage.googleapis.com"
)

var helmBinary = filepath.Join("helm" + moreos.Exe)

type InstallArgs struct {
	PxNamespace  string
	CloudAddress string
	DeployKey    string
}

type Plugin struct {
	*client.Client

	args    *InstallArgs
	options *generic.Options
	// use helm to install pixie, because of the cli of pixie need login and store token in local and karmada issues:
	// https://github.com/karmada-io/karmada/issues/2393, https://github.com/karmada-io/karmada/issues/2392
	helm string
}

func NewPlugin(s *generic.Options, args *InstallArgs) (*Plugin, error) {
	plugin := &Plugin{
		options: s,
		args:    args,
	}
	rest := s.RESTClientGetter()
	c, err := client.NewClient(rest)
	if err != nil {
		return nil, err
	}
	plugin.Client = c
	plugin.helm = "helm"

	return plugin, nil
}

func (p *Plugin) Execute(cmdArgs, environment []string) error {
	if err := p.installHelm(); err != nil {
		return err
	}

	if err := p.addRepo(); err != nil {
		return err
	}

	clusters, err := p.allClusters()
	if err != nil {
		return err
	}

	for _, c := range clusters {
		clusterClient, err := p.Client.NewClusterClientSet(c)
		if err != nil {
			return err
		}

		if _, err := karmadautil.EnsureNamespaceExist(clusterClient, p.args.PxNamespace, p.options.DryRun); err != nil {
			return fmt.Errorf("failed to ensure namespace %s, %w", p.args.PxNamespace, err)
		}

		clusterHelmClient, err := p.Client.NewClusterHelmClient(c)
		if err != nil {
			return err
		}

		_, err = p.applyCrds(clusterHelmClient)
		if err != nil {
			return err
		}

		clusterCRDClient, err := p.Client.NewClusterCRDClientset(c)
		if err != nil {
			return err
		}
		if err := util.WaitCRDReady(clusterCRDClient, vizierCRDName, p.options.WaitInterval, p.options.WaitTimeout); err != nil {
			return fmt.Errorf("wait cluster %s CRD %s ready fail, %w", c, vizierCRDName, err)
		}

		_, err = p.applyTemplates(clusterHelmClient, c)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Plugin) addRepo() error {
	addArgs := []string{
		"repo",
		"add",
		repoName,
		repoAddress,
	}
	addCmd := exec.Command(p.helm, addArgs...)
	err := util.RunCommand(addCmd)
	if err != nil {
		return err
	}

	cmd := exec.Command(p.helm, "repo", "update")
	return util.RunCommand(cmd)
}

func (p *Plugin) applyCrds(helmClient helmclient.Interface) (helmclient.ResourceList, error) {
	args := []string{
		"show",
		"crds",
		fmt.Sprintf("%s/%s", repoName, "pixie-operator-chart"),
	}

	cmd := exec.Command(p.helm, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(string(out))
	}

	r, err := helmClient.Build(bytes.NewBuffer(out), false)
	if err != nil {
		logrus.Debugf("crds: %s", out)
		return nil, fmt.Errorf("failed to build crds: %w", err)
	}

	if _, err := helmClient.Create(r); err != nil {
		return r, err
	}

	return r, nil
}

func (p *Plugin) applyTemplates(helmClient helmclient.Interface, cluster string) (helmclient.ResourceList, error) {
	args := []string{
		"template",
		"--namespace", p.args.PxNamespace,
		fmt.Sprintf("%s/%s", repoName, "pixie-operator-chart"),
		"--set", fmt.Sprintf("clusterName=%s", cluster),
		"--set", fmt.Sprintf("cloudAddr=%s", p.args.CloudAddress),
		"--set", fmt.Sprintf("deployKey=%s", p.args.DeployKey),
	}

	logrus.Debugf("helm template with args: %v", args)
	cmd := exec.Command(p.helm, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(string(out))
	}

	r, err := helmClient.Build(bytes.NewBuffer(out), false)
	if err != nil {
		return nil, err
	}

	r = r.Filter(func(r *resource.Info) bool {
		// crd created in prev steps
		return r.Mapping.GroupVersionKind.Kind != crdKind
	})

	if _, err := helmClient.Create(r); err != nil {
		return r, err
	}

	return r, nil
}

func (p *Plugin) allClusters() ([]string, error) {
	clusters, err := p.KarmadaClient().ClusterV1alpha1().Clusters().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusterNames := make([]string, 0, len(clusters.Items))
	for _, c := range clusters.Items {
		clusterNames = append(clusterNames, c.Name)
	}

	return clusterNames, nil
}

func (p *Plugin) installHelm() error {
	helmComponent := p.options.Components["helm"]

	// TODO: refactor all download code as https://github.com/kurator-dev/kurator/issues/61
	installPath := filepath.Join(p.options.HomeDir, helmComponent.Name, helmComponent.Version)
	helmPath := filepath.Join(installPath, fmt.Sprintf("%s-%s", util.OSExt(), runtime.GOARCH), helmBinary)
	_, err := os.Stat(helmPath)
	if err == nil {
		p.helm = helmPath
		return nil
	}

	if os.IsNotExist(err) {
		if err = os.MkdirAll(installPath, 0o750); err != nil {
			return fmt.Errorf("unable to create directory %q: %w", installPath, err)
		}
		// https://get.helm.sh/helm-v3.9.3-linux-amd64.tar.gz
		url, _ := util.JoinUrlPath(helmComponent.ReleaseURLPrefix,
			fmt.Sprintf("helm-%s-%s-%s.tar.gz", helmComponent.Version, util.OSExt(), runtime.GOARCH))
		if _, err := util.DownloadResource(url, installPath); err != nil {
			return fmt.Errorf("unable to get helm binary %q: %w", installPath, err)
		}
	}

	b, err := util.VerifyExecutableBinary(helmPath)
	if err != nil {
		return err
	}

	p.helm = b
	return err
}
