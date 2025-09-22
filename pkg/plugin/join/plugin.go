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

package join

import (
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"

	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/plugin/karmada"
	"kurator.dev/kurator/pkg/util"
)

type Args struct {
	ClusterName       string
	ClusterKubeConfig string
	ClusterContext    string
}

type JoinPlugin struct {
	options *generic.Options
	args    *Args

	karmadactl string
}

func NewJoinPlugin(o *generic.Options, args *Args) (*JoinPlugin, error) {
	return &JoinPlugin{
		options:    o,
		args:       args,
		karmadactl: "/usr/local/bin/kubectl-karmada",
	}, nil
}

// Execute receives an executable's filepath, a slice
// of arguments, and a slice of environment variables
// to relay to the executable.
func (p *JoinPlugin) Execute(cmdArgs, environment []string) error {
	if err := p.preJoin(); err != nil {
		return err
	}

	if err := p.runJoin(); err != nil {
		logrus.Errorf("failed to join cluster: %v", err)
		return err
	}

	return nil
}

func (p *JoinPlugin) preJoin() error {
	karmadaPlugin, _ := karmada.NewKarmadaPlugin(p.options, nil)
	// download karmadactl
	karmadactlPath, err := karmadaPlugin.InstallKarmadactl()
	if err == nil {
		p.karmadactl = karmadactlPath
	} else {
		logrus.Warnf("install karmadactl failed: %v", err)
	}

	return nil
}

func (p *JoinPlugin) runJoin() error {
	joinArgs := []string{}
	if p.options.KubeConfig != "" {
		joinArgs = append(joinArgs, fmt.Sprintf("--kubeconfig=%s", p.options.KubeConfig))
	}
	joinArgs = append(joinArgs, "join", p.args.ClusterName)
	if p.args.ClusterKubeConfig != "" {
		joinArgs = append(joinArgs, fmt.Sprintf("--cluster-kubeconfig=%s", p.args.ClusterKubeConfig))
	}
	if p.args.ClusterContext != "" {
		joinArgs = append(joinArgs, fmt.Sprintf("--cluster-context=%s", p.args.ClusterContext))
	}
	logrus.Debugf("run cmd: %s %v", p.karmadactl, joinArgs)
	cmd := exec.Command(p.karmadactl, joinArgs...)
	return util.RunCommand(cmd)
}
