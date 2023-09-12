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
	"fmt"
	"time"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	AWS         = "aws"
	HuaWeiCloud = "huaweicloud"
	GCP         = "gcp"
	Azure       = "azure"

	AWSObjStoreSecretName         = "kurator-velero-s3"
	HuaWeiCloudObjStoreSecretName = "kurator-velero-obs"
	GCPObjStoreSecretName         = "kurator-velero-gcs"
	AzureObjStoreSecretName       = "kurator-velero-abs"
)

// reconcileBackupPlugin reconciles the backup plugin configuration and installation across multiple clusters.
// It generates and applies Velero Helm configurations based on the specified backup plugin settings in the fleet specification.
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

	// handle provider-specific details
	objStoreProvider := veleroCfg.Storage.Location.Provider
	// newSecret is a variable used to store the newly created secret object which contains the necessary credentials for the object storage provider. The specific structure and content of the secret vary depending on the provider.
	// providerValues is a map that stores default configurations associated with the specific provider. These configurations are necessary for the proper functioning of the Velero tool with the provider. Currently, this includes configurations for initContainers.
	newSecret, err := f.getProviderDetails(ctx, veleroCfg.Storage.SecretName, objStoreProvider, fleetNN)
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
		}, veleroCfg, newSecret.Name)
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

// getProviderDetails retrieves the secret and provider values based on the specified object storage provider.
func (f *FleetManager) getProviderDetails(ctx context.Context, secretName, objStoreProvider string, fleetNN types.NamespacedName) (*corev1.Secret, error) {
	var newSecret *corev1.Secret
	var err error

	switch objStoreProvider {
	case AWS:
		newSecret, err = f.buildAWSSecret(ctx, secretName, fleetNN)
	case HuaWeiCloud:
		newSecret, err = f.buildHuaWeiCloudSecret(ctx, secretName, fleetNN)
	case GCP:
		newSecret, err = f.buildGCPSecret(ctx, secretName, fleetNN)
	case Azure:
		newSecret, err = f.buildAzureSecret(ctx, secretName, fleetNN)
	default:
		return nil, fmt.Errorf("unknown objStoreProvider: %v", objStoreProvider)
	}

	return newSecret, err
}

// buildAWSSecret constructs a secret for AWS with the necessary credentials.
func (f *FleetManager) buildAWSSecret(ctx context.Context, secretName string, fleetNN types.NamespacedName) (*corev1.Secret, error) {
	// fetch essential information from the user's secret
	accessKey, secretKey, err := getObjStoreCredentials(ctx, f.Client, fleetNN.Namespace, secretName)
	if err != nil {
		return nil, err
	}

	// build an S3 secret for Velero using the accessKey and secretKey
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AWSObjStoreSecretName,
			Namespace: fleetNN.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"cloud": []byte(fmt.Sprintf("[default]\naws_access_key_id=%s\naws_secret_access_key=%s", accessKey, secretKey)),
		},
	}
	return newSecret, nil
}

// TODO： accomplish those function after investigation
func (f *FleetManager) buildHuaWeiCloudSecret(ctx context.Context, secretName string, fleetNN types.NamespacedName) (*corev1.Secret, error) {
	return nil, nil
}
func (f *FleetManager) buildGCPSecret(ctx context.Context, secretName string, fleetNN types.NamespacedName) (*corev1.Secret, error) {
	return nil, nil
}
func (f *FleetManager) buildAzureSecret(ctx context.Context, secretName string, fleetNN types.NamespacedName) (*corev1.Secret, error) {
	return nil, nil
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
