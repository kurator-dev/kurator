/*
Copyright 2022-2025 Kurator Authors.

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

package istio

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"

	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/moreos"
	"kurator.dev/kurator/pkg/util"
)

var istioctlBinary = filepath.Join("istioctl" + moreos.Exe)

type InstallArgs struct {
	Primary string
	Remotes []string

	Cacerts string

	IopFiles []string
	SetFlags []string

	Hub string
	Tag string
}

type IstioPlugin struct {
	*client.Client

	options  *generic.Options
	args     *InstallArgs
	istioctl string
}

const (
	NetworkModeFlat    = "flat"
	NetworkModeNonFlat = "non-flat"
)

func NewIstioPlugin(s *generic.Options, args *InstallArgs) (*IstioPlugin, error) {
	plugin := &IstioPlugin{
		options:  s,
		args:     args,
		istioctl: "/usr/local/bin/istioctl",
	}
	rest := s.RESTClientGetter()
	c, err := client.NewClient(rest)
	if err != nil {
		return nil, err
	}
	plugin.Client = c

	return plugin, nil
}

func (p *IstioPlugin) init() error {
	istioctl, err := p.installIstioctl()
	if err != nil {
		return err
	}
	p.istioctl = istioctl

	return nil
}

// Execute receives an executable's filepath, a slice
// of arguments, and a slice of environment variables
// to relay to the executable.
func (p *IstioPlugin) Execute(cmdArgs, environment []string) error {
	if err := p.init(); err != nil {
		return err
	}

	if err := p.precheck(); err != nil {
		logrus.Infof("istio precheck fail, %s", err)
		return err
	}

	if err := p.runInstall(); err != nil {
		logrus.Infof("failed to install istio, %s", err)
		return err
	}

	return nil
}

func (p *IstioPlugin) installIstioctl() (string, error) {
	istioComponent := p.options.Components["istio"]

	installPath := filepath.Join(p.options.HomeDir, istioComponent.Name, istioComponent.Version)
	istioctlPath := filepath.Join(installPath, istioctlBinary)
	_, err := os.Stat(istioctlPath)
	if err == nil {
		return istioctlPath, nil
	}

	if os.IsNotExist(err) {
		if err = os.MkdirAll(installPath, 0o750); err != nil {
			return "", fmt.Errorf("unable to create directory %q: %w", installPath, err)
		}
		url, _ := util.JoinUrlPath(istioComponent.ReleaseURLPrefix, istioComponent.Version,
			fmt.Sprintf("istioctl-%s-%s-%s.tar.gz", istioComponent.Version, util.OSExt(), runtime.GOARCH))
		if _, err := util.DownloadResource(url, installPath); err != nil {
			return "", fmt.Errorf("unable to get istioctl binary %q: %w", installPath, err)
		}
	}

	return util.VerifyExecutableBinary(istioctlPath)
}
