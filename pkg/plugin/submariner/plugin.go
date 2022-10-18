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
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"

	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/moreos"
	"kurator.dev/kurator/pkg/util"
)

var subctlBinary = filepath.Join("subctl" + moreos.Exe)

type InstallArgs struct {
}

type SubmarinerPlugin struct {
	*client.Client

	options     *generic.Options
	args        *InstallArgs
	subctl      string
	installPath string
}

func NewSubmarinerPlugin(s *generic.Options, args *InstallArgs) (*SubmarinerPlugin, error) {
	plugin := &SubmarinerPlugin{
		options: s,
		args:    args,
		subctl:  "/usr/local/bin/subctl",
	}
	rest := s.RESTClientGetter()
	c, err := client.NewClient(rest)
	if err != nil {
		return nil, err
	}
	plugin.Client = c

	return plugin, nil
}

func (p *SubmarinerPlugin) init() error {
	subctl, err := p.installSubctl()
	if err != nil {
		return err
	}
	p.subctl = subctl

	return nil
}

// Execute receives an executable's filepath, a slice
// of arguments, and a slice of environment variables
// to relay to the executable.
func (p *SubmarinerPlugin) Execute(cmdArgs, environment []string) error {
	if err := p.init(); err != nil {
		return err
	}

	if err := p.runInstall(); err != nil {
		logrus.Infof("failed to install submariner, %s", err)
		return err
	}

	return nil
}

func (p *SubmarinerPlugin) installSubctl() (string, error) {
	istioComponent := p.options.Components["submariner"]
	p.installPath = filepath.Join(p.options.HomeDir, istioComponent.Name, istioComponent.Version)
	subctlPath := filepath.Join(p.installPath, subctlBinary)
	_, err := os.Stat(subctlPath)
	if err == nil {
		return subctlPath, nil
	}

	if os.IsNotExist(err) {
		if err = os.MkdirAll(p.installPath, 0o750); err != nil {
			return "", fmt.Errorf("unable to create directory %q: %w", p.installPath, err)
		}
		// https://github.com/submariner-io/releases/releases/download/v0.12.2/subctl-v0.12.2-linux-amd64.tar.xz
		// For other tar format, we use `tar` command directly
		subctl := fmt.Sprintf("subctl-%s-%s-%s", istioComponent.Version, util.OSExt(), runtime.GOARCH)
		pkg := subctl + ".tar.xz"
		url, _ := util.JoinUrlPath(istioComponent.ReleaseURLPrefix, istioComponent.Version, pkg)
		fileName := path.Join(p.installPath, pkg)
		// download subctl tarball
		if _, err := util.DownloadResource(url, fileName); err != nil {
			return "", fmt.Errorf("unable to get subctl binary %q: %w", p.installPath, err)
		}
		cmd := exec.Command("tar", "-xvf", fileName, "--strip-components", "1", "-C", p.installPath)
		err := util.RunCommand(cmd)
		if err != nil {
			return "", err
		}
		subctlPath = path.Join(p.installPath, subctl)
	}

	return util.VerifyExecutableBinary(subctlPath)
}
