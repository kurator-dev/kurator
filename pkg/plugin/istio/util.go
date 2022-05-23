package istio

import (
	"github.com/zirain/ubrain/pkg/client"
	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/util"
)

func waitIngressgatewayReady(client *client.Client, opts *generic.Options, cluster string) error {
	return util.WaitKarmadaClusterPodReady(client, cluster, istioSystemNamespace, "app=istio-ingressgateway", opts.WaitInterval, opts.WaitTimeout)
}
