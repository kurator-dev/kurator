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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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

	AWSObjStoreSecretNameSuffix         = "-velero-s3"
	HuaWeiCloudObjStoreSecretNameSuffix = "-velero-obs"
	GCPObjStoreSecretNameSuffix         = "-velero-gcs"
	AzureObjStoreSecretNameSuffix       = "-velero-abs"

	ObjStoreSecretNamespace = "velero"
)

// reconcileBackupPlugin reconciles the backup plugin configuration and installation across multiple clusters.
// It generates and applies Velero Helm configurations based on the specified backup plugin settings in the fleet specification.
func (f *FleetManager) reconcileBackupPlugin(ctx context.Context, fleet *v1alpha1.Fleet, fleetClusters map[ClusterKey]*FleetCluster) (kube.ResourceList, ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	veleroCfg := fleet.Spec.Plugin.Backup

	if veleroCfg == nil {
		// reconcilePluginResources will delete all resources if plugin is nil
		return nil, ctrl.Result{}, nil
	}

	fleetNN := types.NamespacedName{
		Namespace: fleet.Namespace,
		Name:      fleet.Name,
	}

	// handle provider-specific details
	objStoreProvider := veleroCfg.Storage.Location.Provider
	// newSecret is a variable used to store the newly created secret object which contains the necessary credentials for the object storage provider. The specific structure and content of the secret vary depending on the provider.
	newSecret, err := f.buildNewSecret(ctx, veleroCfg.Storage.SecretName, objStoreProvider, fleetNN)
	if err != nil {
		err = fmt.Errorf("error building new secret for objStoreProvider %s: %w", objStoreProvider, err)
		return nil, ctrl.Result{}, err
	}

	fleetOwnerRef := ownerReference(fleet)
	var resources kube.ResourceList

	// Iterating through each fleet cluster to generate and apply Velero helm configurations.
	for key, cluster := range fleetClusters {
		// generate Velero helm config for each fleet cluster
		b, err := plugin.RenderVelero(f.Manifests, fleetNN, fleetOwnerRef, plugin.FleetCluster{
			Name:       key.Name,
			SecretName: cluster.Secret,
			SecretKey:  cluster.SecretKey,
		}, veleroCfg, newSecret.Name)
		if err != nil {
			err = fmt.Errorf("error rendering Velero for fleet cluster %s: %w", key.Name, err)
			return nil, ctrl.Result{}, err
		}

		// create a new secret in the current fleet cluster before initializing the backup plugin.
		if err := createNewSecretInFleetCluster(ctx, cluster, newSecret); err != nil {
			err = fmt.Errorf("error creating new secret in fleet cluster %s: %w", key.Name, err)
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

	// After the HelmRelease reconciliation, we update the owner references of the new secrets to point to the Velero deployment.
	// This ensures that when the Velero deployment (created by Kurator) is deleted, its associated secrets are cleaned up automatically,
	// preventing orphaned resources and maintaining the cleanliness of the cluster.
	for key, cluster := range fleetClusters {
		if err := f.updateNewSecretOwnerReference(ctx, key.Name, cluster, newSecret); err != nil {
			err = fmt.Errorf("error updating owner reference for secret in cluster %s: %w", key.Name, err)
			return nil, ctrl.Result{}, err
		}
	}

	return resources, ctrl.Result{}, nil
}

// buildNewSecret generate a new secret for Velero based on the specified object storage provider.
func (f *FleetManager) buildNewSecret(ctx context.Context, secretName, objStoreProvider string, fleetNN types.NamespacedName) (*corev1.Secret, error) {
	var newSecret *corev1.Secret
	var err error

	switch objStoreProvider {
	case AWS:
		newSecret, err = f.buildAWSSecret(ctx, secretName, fleetNN, fleetNN.Name+AWSObjStoreSecretNameSuffix)
	case HuaWeiCloud:
		newSecret, err = f.buildHuaWeiCloudSecret(ctx, secretName, fleetNN, fleetNN.Name+HuaWeiCloudObjStoreSecretNameSuffix)
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
func (f *FleetManager) buildAWSSecret(ctx context.Context, secretName string, fleetNN types.NamespacedName, s3SecretName string) (*corev1.Secret, error) {
	// fetch essential information from the user's secret
	accessKey, secretKey, err := getObjStoreCredentials(ctx, f.Client, fleetNN.Namespace, secretName)
	if err != nil {
		return nil, err
	}

	// build an S3 secret for Velero using the accessKey and secretKey
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3SecretName,
			Namespace: ObjStoreSecretNamespace,
			Labels: map[string]string{
				FleetPluginName: plugin.BackupPluginName,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"cloud": []byte(fmt.Sprintf("[default]\naws_access_key_id=%s\naws_secret_access_key=%s", accessKey, secretKey)),
		},
	}
	return newSecret, nil
}

func (f *FleetManager) buildHuaWeiCloudSecret(ctx context.Context, secretName string, fleetNN types.NamespacedName, huaweiyunSecretName string) (*corev1.Secret, error) {
	return f.buildAWSSecret(ctx, secretName, fleetNN, huaweiyunSecretName)
}

// TODO： accomplish those function after investigation
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

// createNewSecretInFleetCluster creates a new secret in the specified fleet cluster.
// It takes a FleetCluster instance and a pre-built corev1.Secret instance as parameters.
// It uses the kube client from the FleetCluster instance to create the new secret in the respective cluster.
func createNewSecretInFleetCluster(ctx context.Context, cluster *FleetCluster, newSecret *corev1.Secret) error {
	// Get the kubeclient.Interface instance
	kubeClient := cluster.client.CtrlRuntimeClient()

	// Get the namespace of the secret
	namespace := newSecret.Namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	// Create or update namespace
	if _, syncErr := controllerutil.CreateOrUpdate(ctx, kubeClient, ns, func() error {
		return nil
	}); syncErr != nil {
		return fmt.Errorf("failed to sync namespace %s: %w", namespace, syncErr)
	}

	// Create or update new secret
	if _, syncErr := controllerutil.CreateOrUpdate(ctx, kubeClient, newSecret, func() error {
		return nil
	}); syncErr != nil {
		return fmt.Errorf("failed to sync new secret in namespace %s: %w", namespace, syncErr)
	}

	return nil
}

// updateNewSecretOwnerReference updates the OwnerReferences of the given secret in the specified fleet cluster.
func (f *FleetManager) updateNewSecretOwnerReference(ctx context.Context, clusterName string, cluster *FleetCluster, newSecret *corev1.Secret) error {
	// Get the kubeclient.Interface instance
	kubeClient := cluster.client.KubeClient()

	veleroDeploymentName := getVeleroDeploymentName(clusterName)
	veleroDeployment, err := kubeClient.AppsV1().Deployments(newSecret.Namespace).Get(ctx, veleroDeploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Set the OwnerReferences of the Secret to the Velero Deployment
	newSecret.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       veleroDeployment.Name,
			UID:        veleroDeployment.UID,
		},
	}

	// Update the Secret object with the new OwnerReferences
	if _, err := kubeClient.CoreV1().Secrets(newSecret.Namespace).Update(ctx, newSecret, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

// getVeleroDeploymentName returns the formatted deployment name in the current cluster.
// The name is constructed as follows: "velero-" + "ComponentName" + "-" + clusterName.
func getVeleroDeploymentName(clusterName string) string {
	return fmt.Sprintf("velero-%s-%s", plugin.VeleroComponentName, clusterName)
}
