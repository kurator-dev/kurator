package istio

import (
	"github.com/sirupsen/logrus"
	"github.com/zirain/ubrain/pkg/client"
	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/util"
)

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

	settings *generic.Options
	args     *InstallArgs
	getter   *util.BinaryGetter

	istioctl string
}

func NewIstioPlugin(s *generic.Options, args *InstallArgs) (*IstioPlugin, error) {
	plugin := &IstioPlugin{
		settings: s,
		args:     args,
		getter:   util.NewBinaryGetter(s),
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
	binaryPath, err := p.getter.Istioctl()
	if err != nil {
		return err
	}
	p.istioctl = binaryPath

	return nil
}

// Execute receives an executable's filepath, a slice
// of arguments, and a slice of environment variables
// to relay to the executable.
func (p *IstioPlugin) Execute(cmdArgs, environment []string) error {
	if err := p.init(); err != nil {
		return err
	}

	if err := p.runInstall(); err != nil {
		logrus.Infof("failed to install istio, %s", err)
		return err
	}

	return nil
}
