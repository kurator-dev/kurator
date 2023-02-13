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
	"context"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	infrav1 "kurator.dev/kurator/pkg/apis/infra/v1alpha1"
	"kurator.dev/kurator/pkg/controllers/scope"
	"kurator.dev/kurator/pkg/controllers/template"
)

func (r *ClusterController) reconcileCNI(ctx context.Context, infraCluster *infrav1.Cluster) (ctrl.Result, error) {
	// For now, use CusterResourceSet to apply the CNI resources
	cni, err := template.RenderCNI(scope.CNI{
		Name:      infraCluster.Name,
		Namespace: infraCluster.Namespace,
		Type:      infraCluster.Spec.Network.CNI.Type,
	})
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to render CNI resources")
	}

	_, err = patchResources(ctx, cni)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to apply CNI resources")
	}

	return ctrl.Result{}, nil
}
