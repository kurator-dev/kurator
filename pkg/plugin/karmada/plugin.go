package karmada

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"

	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/moreos"
	"github.com/zirain/ubrain/pkg/util"
)

var karmadactlBinary = filepath.Join("kubectl-karmada" + moreos.Exe)

type KarmadaPlugin struct {
	options    *generic.Options
	karmadactl string
}

func NewKarmadaPlugin(o *generic.Options) (*KarmadaPlugin, error) {
	return &KarmadaPlugin{
		options:    o,
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
		logrus.Errorf("failed to install karmada, %s", err)
		return err
	}

	return nil
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
