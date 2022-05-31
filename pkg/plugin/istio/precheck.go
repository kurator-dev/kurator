package istio

import (
	"errors"

	"kurator.dev/kurator/pkg/util"
)

func (p *IstioPlugin) precheck() error {
	if len(p.args.Primary) == 0 {
		return errors.New("must provide a cluster to install istio primary")
	}

	return util.IsClustersReady(p.KarmadaClient(), p.allClusters()...)
}
