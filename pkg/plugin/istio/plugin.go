package istio

import (
	"fmt"
	"time"

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

type IstioPluginHandler struct {
	client *util.Client

	settings *generic.Options
	args     *InstallArgs
	getter   *util.BinaryGetter

	istioctl string
}

func NewIstioPluginHandler(s *generic.Options, args *InstallArgs) (*IstioPluginHandler, error) {
	plugin := &IstioPluginHandler{
		settings: s,
		args:     args,
		getter:   util.NewBinaryGetter(s),
		istioctl: "/usr/local/bin/istioctl",
	}
	rest := s.RESTClientGetter()
	c, err := util.NewClient(rest)
	if err != nil {
		return nil, err
	}
	plugin.client = c

	return plugin, nil
}

func (plugin *IstioPluginHandler) init() error {
	binaryPath, err := plugin.getter.Istioctl()
	if err != nil {
		return err
	}
	plugin.istioctl = binaryPath

	return nil
}

// Execute receives an executable's filepath, a slice
// of arguments, and a slice of environment variables
// to relay to the executable.
func (plugin *IstioPluginHandler) Execute(cmdArgs, environment []string) error {
	if err := plugin.init(); err != nil {
		return err
	}

	if err := plugin.runInstall(); err != nil {
		plugin.Infof("failed to install istio, %s", err)
		return err
	}

	return nil
}

func (plugin *IstioPluginHandler) Infof(format string, a ...interface{}) {
	if plugin.settings.Ui == nil {
		return
	}
	plugin.settings.Ui.Output(fmt.Sprintf("%s\t%s\t", time.Now().Format("2006-01-02 15:04:05"), "[Istio]") + fmt.Sprintf(format, a...))
}
