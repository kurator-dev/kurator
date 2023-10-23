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
	"net"
	"reflect"
	"time"

	hrapiv2b1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"helm.sh/helm/v3/pkg/kube"
	"istio.io/istio/pkg/util/sets"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	capiutil "sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"

	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	"kurator.dev/kurator/pkg/fleet-manager/plugin"
	"kurator.dev/kurator/pkg/infra/util"
)

func (f *FleetManager) reconcileObjStoreSecretOwnerReference(ctx context.Context, fleet *fleetapi.Fleet, fleetClusters map[ClusterKey]*fleetCluster) error {
	for _, cluster := range fleet.Spec.Clusters {
		fleetCluster, ok := fleetClusters[ClusterKey{cluster.Kind, cluster.Name}]
		if !ok {
			// can happen when the cluster is not present
			continue
		}

		// reconcile objstore secret's owner reference
		// a statefulset named prometheus-prometheus-prometheus is created by HelmRelease in each cluster
		sts, err := fleetCluster.client.KubeClient().AppsV1().StatefulSets(MonitoringNamespace).Get(ctx, "prometheus-prometheus-prometheus", metav1.GetOptions{})
		if err != nil {
			return err
		}

		secret, err := fleetCluster.client.KubeClient().CoreV1().Secrets(MonitoringNamespace).Get(ctx, fleet.Spec.Plugin.Metric.Thanos.ObjectStoreConfig.SecretName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		stsOwnerReference := metav1.OwnerReference{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet", // TODO: use typemeta package
			Name:       sts.Name,
			UID:        sts.UID,
		}
		if !capiutil.HasOwnerRef(secret.OwnerReferences, stsOwnerReference) {
			secret.OwnerReferences = append(secret.OwnerReferences, stsOwnerReference)
			if _, err := fleetCluster.client.KubeClient().CoreV1().Secrets(MonitoringNamespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
				return err
			}
		}
	}
	return nil
}

// reconcileSidecarRemoteService reconciles a headless service named thanos-sidecar-remote, and ensure owner reference is set for all resources
// TODO: find a better way to collect service endpoints of thanos-sidecar-remote service after all helm releases are reconciled
func (f *FleetManager) reconcileSidecarRemoteService(ctx context.Context, fleet *fleetapi.Fleet, fleetClusters map[ClusterKey]*fleetCluster) error {
	log := ctrl.LoggerFrom(ctx)
	log = log.WithValues("fleet", types.NamespacedName{Name: fleet.Name, Namespace: fleet.Namespace})

	endpoints := sets.New[string]()
	for _, cluster := range fleet.Spec.Clusters {
		fleetCluster, ok := fleetClusters[ClusterKey{cluster.Kind, cluster.Name}]
		if !ok {
			// it can happen when the cluster is not present
			continue
		}

		svc, err := fleetCluster.client.KubeClient().CoreV1().Services(MonitoringNamespace).Get(ctx, PrometheusThanosServiceName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
			continue
		}

		for _, lb := range svc.Status.LoadBalancer.Ingress {
			// TODO: There is a case when lb has not assigned an ip to the service, if we choice to requeue the fleet,
			// the fleet will never became ready in the worest case.
			if net.ParseIP(lb.IP) == nil {
				// skip invalid ip
				continue
			}

			endpoints.Insert(lb.IP)
		}
	}

	thanosHelmRelease := &hrapiv2b1.HelmRelease{}
	thanosHelmReleaseNN := types.NamespacedName{Namespace: fleet.Namespace, Name: "thanos"}
	if err := f.Client.Get(context.Background(), thanosHelmReleaseNN, thanosHelmRelease); err != nil {
		log.Error(err, "failed to get thanos helm release")
		return err
	}

	ownerReference := metav1.OwnerReference{
		APIVersion: "helm.toolkit.fluxcd.io/v2beta1",
		Kind:       hrapiv2b1.HelmReleaseKind,
		Name:       thanosHelmRelease.Name,
		UID:        thanosHelmRelease.UID,
	}

	svc := &corev1.Service{}
	svcNN := types.NamespacedName{Namespace: fleet.Namespace, Name: "thanos-sidecar-remote"}
	if err := f.Client.Get(context.Background(), svcNN, svc); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		svc = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcNN.Name,
				Namespace: svcNN.Namespace,
				Labels:    fleetMetricResourceLabels(fleet.Name),
				OwnerReferences: []metav1.OwnerReference{
					ownerReference,
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name:        "grpc",
						Port:        10901,
						Protocol:    corev1.ProtocolTCP,
						AppProtocol: pointer.String("grpc"),
					},
				},
				ClusterIP: NoneClusterIP,
			},
		}

		if err := f.Client.Create(context.Background(), svc); err != nil {
			return err
		}
	}

	if !capiutil.HasOwnerRef(svc.OwnerReferences, ownerReference) {
		svc.OwnerReferences = append(svc.OwnerReferences, ownerReference)
		if err := f.Client.Update(context.Background(), svc); err != nil {
			return err
		}
	}

	ep := &corev1.Endpoints{}
	if err := f.Client.Get(context.Background(), svcNN, ep); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		ep = &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcNN.Name,
				Namespace: svcNN.Namespace,
			},
			Subsets: convertToSubset(endpoints),
		}

		if err := f.Client.Create(context.Background(), ep); err != nil {
			return err
		}
	}

	subsets := convertToSubset(endpoints)
	if reflect.DeepEqual(ep.Subsets, subsets) {
		return nil
	}

	ep.Subsets = subsets
	if err := f.Client.Update(context.Background(), ep); err != nil {
		return err
	}

	return nil
}

// syncObjStoreSecret syncs the secret to the cluster
func (f *FleetManager) syncObjStoreSecret(ctx context.Context, fleetCluster *fleetCluster, secret *corev1.Secret) error {
	_, err := fleetCluster.client.KubeClient().CoreV1().Namespaces().Get(ctx, secret.Namespace, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err := fleetCluster.client.KubeClient().CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: secret.Namespace,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return nil
		}
	} else if err != nil {
		return nil
	}

	s, err := fleetCluster.client.KubeClient().CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err := fleetCluster.client.KubeClient().CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	s.Data = secret.Data
	_, err = fleetCluster.client.KubeClient().CoreV1().Secrets(secret.Namespace).Update(ctx, s, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (f *FleetManager) reconcileMetricPlugin(ctx context.Context, fleet *fleetapi.Fleet, fleetClusters map[ClusterKey]*fleetCluster) (kube.ResourceList, ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if fleet.Spec.Plugin.Metric == nil {
		// reconcilePluginResources will delete all resources if plugin is nil
		return nil, ctrl.Result{}, nil
	}

	fleetNN := types.NamespacedName{
		Namespace: fleet.Namespace,
		Name:      fleet.Name,
	}
	metricCfg := fleet.Spec.Plugin.Metric
	fleetOwnerRef := ownerReference(fleet)

	b, err := plugin.RenderThanos(f.Manifests, fleetNN, fleetOwnerRef, metricCfg)
	if err != nil {
		return nil, ctrl.Result{}, err
	}

	thanosResources, err := util.PatchResources(b)
	if err != nil {
		return nil, ctrl.Result{}, err
	}
	var resources kube.ResourceList
	resources = append(resources, thanosResources...)

	// prepare objstore secret for fleet cluster
	objSecret := &corev1.Secret{}
	if err := f.Client.Get(ctx, types.NamespacedName{Namespace: fleet.Namespace, Name: metricCfg.Thanos.ObjectStoreConfig.SecretName}, objSecret); err != nil {
		return nil, ctrl.Result{}, err
	}
	promSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      objSecret.Name,
			Namespace: MonitoringNamespace, // TODO: make it configurable
			Labels:    fleetMetricResourceLabels(fleet.Name),
		},
		Data: objSecret.Data,
	}

	log.V(4).Info("start to reconcile prometheus plugin for every cluster in fleet")
	for _, c := range fleet.Spec.Clusters {
		fleetCluster, ok := fleetClusters[ClusterKey{c.Kind, c.Name}]
		if !ok {
			// could happen when cluster not present
			continue
		}

		// TODO: find a better way to sync objstore secret to member clusters
		if err := f.syncObjStoreSecret(ctx, fleetCluster, promSecret); err != nil {
			return nil, ctrl.Result{}, fmt.Errorf("failed to reconcile objstore secret for cluster %s: %w", c.Name, err)
		}

		b, err := plugin.RenderPrometheus(f.Manifests, fleetNN, fleetOwnerRef, plugin.FleetCluster{
			Name:       c.Name,
			SecretName: fleetCluster.Secret,
			SecretKey:  fleetCluster.SecretKey,
		}, metricCfg)
		if err != nil {
			return nil, ctrl.Result{}, err
		}

		// apply HelmRepository and HelmRelease for prometheus per cluster
		prometheusResources, err := util.PatchResources(b)
		if err != nil {
			return nil, ctrl.Result{}, err
		}
		resources = append(resources, prometheusResources...)
	}

	log.V(4).Info("wait for helm release to be reconciled")
	if !f.helmReleaseReady(ctx, fleet, resources) {
		// wait for HelmRelease to be ready
		return nil, ctrl.Result{
			// HelmRelease check interval is 1m, so we set 30s here
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	log.V(4).Info("begin to reconcile owner reference for metric plugin")
	if err := f.reconcileObjStoreSecretOwnerReference(ctx, fleet, fleetClusters); err != nil {
		return nil, ctrl.Result{}, err
	}

	log.V(4).Info("begin to reconcile sidecar remote service for metric plugin")
	if err := f.reconcileSidecarRemoteService(ctx, fleet, fleetClusters); err != nil {
		return nil, ctrl.Result{}, err
	}

	return resources, ctrl.Result{}, nil
}
