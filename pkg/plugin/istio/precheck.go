package istio

import (
	"errors"

	"github.com/zirain/ubrain/pkg/util"
)

func (p *IstioPlugin) precheck() error {
	if len(p.args.Primary) == 0 {
		return errors.New("must provide a cluster to install istio primary")
	}

	return util.IsClustersReady(p.KarmadaClient(), p.allClusters()...)
}
