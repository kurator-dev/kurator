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

package karmada

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"

	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/moreos"
	"kurator.dev/kurator/pkg/util"
)

var karmadactlBinary = filepath.Join("kubectl-karmada" + moreos.Exe)

const (
	karmadaSystemNamespace      = "karmada-system"
	aggregatedApiserverSelector = "app=karmada-aggregated-apiserver"
)

type KarmadaPlugin struct {
	*client.Client

	options    *generic.Options
	karmadactl string
}

func NewKarmadaPlugin(o *generic.Options) (*KarmadaPlugin, error) {
	p := &KarmadaPlugin{
		options:    o,
		karmadactl: "/usr/local/bin/kubectl-karmada",
	}

	rest := o.RESTClientGetter()
	c, err := client.NewClient(rest)
	if err != nil {
		return nil, err
	}
	p.Client = c

	return p, nil
}

// Execute receives an executable's filepath, a slice
// of arguments, and a slice of environment variables
// to relay to the executable.
func (p *KarmadaPlugin) Execute(cmdArgs, environment []string) error {
	if err := p.preInstall(); err != nil {
		return err
	}

	if err := p.runInstall(); err != nil {
		logrus.Errorf("failed to install karmada, %s", err)
		return err
	}

	// make sure karmada-aggregated-apiserver is ready, https://github.com/karmada-io/karmada/issues/1836
	return util.WaitPodReady(p.KubeClient(), karmadaSystemNamespace, aggregatedApiserverSelector, p.options.WaitInterval, p.options.WaitTimeout)
}

func (p *KarmadaPlugin) preInstall() error {
	// install karmadactl
	karmadactlPath, err := p.InstallKarmadactl()
	if err == nil {
		p.karmadactl = karmadactlPath
	} else {
		logrus.Warnf("install karmadactl failed: %v", err)
	}

	return nil
}

func (p *KarmadaPlugin) runInstall() error {
	installArgs := []string{
		"init",
	}
	if p.options.KubeConfig != "" {
		installArgs = append(installArgs, fmt.Sprintf("--kubeconfig=%s", p.options.KubeConfig))
	}
	logrus.Debugf("run cmd: %s %v", p.karmadactl, installArgs)
	cmd := exec.Command(p.karmadactl, installArgs...)
	err := util.RunCommand(cmd)
	return err
}

func (p *KarmadaPlugin) InstallKarmadactl() (string, error) {
	karmadaComponent := p.options.Components["karmada"]
	installPath := filepath.Join(p.options.HomeDir, karmadaComponent.Name, karmadaComponent.Version)
	karmadactlPath := filepath.Join(installPath, karmadactlBinary)
	_, err := os.Stat(karmadactlPath)
	if err == nil {
		return karmadactlPath, nil
	}

	if os.IsNotExist(err) {
		if err = os.MkdirAll(installPath, 0o750); err != nil {
			return "", fmt.Errorf("unable to create directory %q: %w", installPath, err)
		}
		url, _ := util.JoinUrlPath(karmadaComponent.ReleaseURLPrefix, karmadaComponent.Version,
			fmt.Sprintf("kubectl-karmada-%s-%s.tgz", util.OSExt(), runtime.GOARCH))
		if _, err = util.DownloadResource(url, installPath); err != nil {
			return "", fmt.Errorf("unable to get karmadactl binary %q: %w", url, err)
		}
	}
	return util.VerifyExecutableBinary(karmadactlPath)
}
