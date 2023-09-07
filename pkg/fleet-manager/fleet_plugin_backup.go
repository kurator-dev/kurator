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

package fleet

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	"kurator.dev/kurator/pkg/fleet-manager/plugin"
	"kurator.dev/kurator/pkg/infra/util"
)

const (
	AccessKey = "access-key"
	SecretKey = "secret-key"
)

func (f *FleetManager) reconcileBackupPlugin(ctx context.Context, fleet *v1alpha1.Fleet, fleetClusters map[ClusterKey]*fleetCluster) (kube.ResourceList, ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if fleet.Spec.Plugin.Backup == nil {
		// reconcilePluginResources will delete all resources if plugin is nil
		return nil, ctrl.Result{}, nil
	}

	veleroCfg := fleet.Spec.Plugin.Backup

	fleetNN := types.NamespacedName{
		Namespace: fleet.Namespace,
		Name:      fleet.Name,
	}

	// fetch accessKey and secretKey from secret.
	accessKey, secretKey, err := getObjStoreCredentials(ctx, f.Client, fleet.Namespace, fleet.Spec.Plugin.Backup.Storage.SecretName)
	if err != nil {
		return nil, ctrl.Result{}, err
	}
	fleetOwnerRef := ownerReference(fleet)
	var resources kube.ResourceList
	for key, cluster := range fleetClusters {
		// generate Velero helm config for each fleet cluster
		b, err := plugin.RenderVelero(f.Manifests, fleetNN, fleetOwnerRef, plugin.FleetCluster{
			Name:       key.Name,
			SecretName: cluster.Secret,
			SecretKey:  cluster.SecretKey,
		}, veleroCfg, accessKey, secretKey)
		if err != nil {
			return nil, ctrl.Result{}, err
		}

		// apply Velero helm resources
		veleroResources, err := util.PatchResources(b)
		if err != nil {
			return nil, ctrl.Result{}, err
		}
		resources = append(resources, veleroResources...)
	}

	log.V(4).Info("wait for velero helm release to be reconciled")
	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	return resources, ctrl.Result{}, nil
}

func getObjStoreCredentials(ctx context.Context, client client.Client, namespace, secretName string) (accessKey, secretKey string, err error) {
	secret := &corev1.Secret{}
	SecretNN := types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}

	if err := client.Get(ctx, SecretNN, secret); err != nil {
		return "", "", errors.Wrapf(err, "failed to get cluster secret %s in namespace %s", secretName, namespace)
	}

	accessKey = string(secret.Data[AccessKey])
	secretKey = string(secret.Data[SecretKey])

	return accessKey, secretKey, nil
}
