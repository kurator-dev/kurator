package kubeedge

import (
	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/util"
)

func waitCloudcoreReady(client *client.Client, opts *generic.Options, cluster, namespace string) error {
	return util.WaitMemberClusterPodReady(client, cluster, namespace, "kubeedge=cloudcore", opts.WaitInterval, opts.WaitTimeout)
}
