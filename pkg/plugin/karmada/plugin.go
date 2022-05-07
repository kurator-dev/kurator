package karmada

import (
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/util"
)

type KarmadaPlugin struct {
	options *generic.Options
	getter  *util.BinaryGetter

	karmadactl string
}

func NewKarmadaPlugin(o *generic.Options) (*KarmadaPlugin, error) {
	return &KarmadaPlugin{
		options:    o,
		getter:     util.NewBinaryGetter(o),
		karmadactl: "/usr/local/bin/kubectl-karmada",
	}, nil
}

// Execute receives an executable's filepath, a slice
// of arguments, and a slice of environment variables
// to relay to the executable.
func (p *KarmadaPlugin) Execute(cmdArgs, environment []string) error {
	if err := p.preInstall(); err != nil {
		return err
	}

	if err := p.runInstall(); err != nil {
		logrus.Errorf("failed to install istio, %s", err)
		return err
	}

	return nil
}

func (p *KarmadaPlugin) preInstall() error {
	// download karmadactl
	karmadactlPath, err := p.getter.Karmadactl()
	if err == nil {
		p.karmadactl = karmadactlPath
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
	out, err := cmd.CombinedOutput()
	logrus.Infof("%s", string(out))
	return err
}
