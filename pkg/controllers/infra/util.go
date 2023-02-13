/*
Copyright Kurator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package infra

import (
	"bytes"
	"context"
	"time"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/kube"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "kurator.dev/kurator/pkg/apis/infra/v1alpha1"
	"kurator.dev/kurator/pkg/client"
)

func patchResources(ctx context.Context, b []byte) (kube.ResourceList, error) {
	rest, err := ctrl.GetConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get kubeconfig")
	}
	c, err := client.NewClient(client.NewRESTClientGetter(rest))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create client")
	}
	target, err := c.HelmClient().Build(bytes.NewBuffer(b), false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build resources")
	}
	if _, err := c.HelmClient().Update(target, target, true); err != nil {
		return nil, errors.Wrapf(err, "failed to update resources")
	}
	if err := c.HelmClient().Wait(target, time.Minute); err != nil {
		return nil, errors.Wrapf(err, "failed to wait for resources")
	}

	return target, nil
}

const (
	clusterNameLabel      = "infra.kurator.dev/cluster-name"
	clusterNamespaceLabel = "infra.kurator.dev/cluster-namespace"
)

func clusterMatchingLabels(infraCluster *infrav1.Cluster) ctrlclient.MatchingLabels {
	return ctrlclient.MatchingLabels{
		clusterNameLabel:      infraCluster.Name,
		clusterNamespaceLabel: infraCluster.Namespace,
	}
}
