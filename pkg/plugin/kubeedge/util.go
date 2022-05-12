package kubeedge

import (
	"github.com/zirain/ubrain/pkg/client"
	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/util"
)

func waitCloudcoreReady(client *client.Client, opts *generic.Options, cluster, namespace string) error {
	return util.WaitPodReady(client, cluster, namespace, "kubeedge=cloudcore", opts.WaitInterval, opts.WaitTimeout)
}
