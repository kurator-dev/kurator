package istio

import (
	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/util"
)

func waitIngressgatewayReady(client *client.Client, opts *generic.Options, cluster string) error {
	return util.WaitMemberClusterPodReady(client, cluster, istioSystemNamespace, "app=istio-ingressgateway", opts.WaitInterval, opts.WaitTimeout)
}
