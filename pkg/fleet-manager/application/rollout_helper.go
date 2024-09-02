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

package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	flaggerv1b1 "github.com/fluxcd/flagger/pkg/apis/flagger/v1beta1"
	"github.com/pkg/errors"
	"istio.io/istio/pkg/util/sets"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ingressv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	applicationapi "kurator.dev/kurator/pkg/apis/apps/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	fleetmanager "kurator.dev/kurator/pkg/fleet-manager"
	render "kurator.dev/kurator/pkg/fleet-manager/application/manifests"
	plugin "kurator.dev/kurator/pkg/fleet-manager/plugin"
)

const (
	// kurator rollout labels
	RolloutIdentifier = "kurator.dev/rollout"
	istioInject       = "istio-injection"
	kumaInject        = "kuma.io/sidecar-injection"
	// StatusSyncInterval specifies the interval for requeueing when synchronizing status. It determines how frequently the status should be checked and updated.
	StatusSyncInterval = 30 * time.Second

	currentClusterKind = "currentCluster"
	currentClusterName = "host"
	// resources config
	ingressAPIVersion      = "networking.k8s.io/v1"
	ingressKind            = "Ingress"
	ingressName            = "nginx"
	ingressLabelKey        = "app"
	ingressAnnotationKey   = "kubernetes.io/ingress.class"
	ingressAnnotationValue = "nginx"

	kumaAnnotation = "9898.service.kuma.io/protocol"
)

func (a *ApplicationManager) fetchRolloutClusters(ctx context.Context,
	app *applicationapi.Application,
	kubeClient client.Client,
	fleet *fleetapi.Fleet,
	syncPolicy *applicationapi.ApplicationSyncPolicy,
) (map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster, error) {
	log := ctrl.LoggerFrom(ctx)
	var fleetclusters map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster
	if fleet == nil {
		fleetclusters = make(map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster, 1)
		client, err := fleetmanager.WrapClient(a.Client)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to wrap client")
		}
		fleetclusters[fleetmanager.ClusterKey{Kind: currentClusterKind, Name: currentClusterName}] = &fleetmanager.FleetCluster{
			Client: client,
		}
	} else {
		destination := getPolicyDestination(app, syncPolicy)
		ClusterInterfaceList, result, err := a.fetchFleetClusterList(ctx, fleet, destination.ClusterSelector)
		if err != nil || result.RequeueAfter > 0 {
			return nil, err
		}

		fleetclusters = make(map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster, len(ClusterInterfaceList))
		for _, cluster := range ClusterInterfaceList {
			kclient, err := fleetmanager.ClientForCluster(kubeClient, fleet.Namespace, cluster)
			if err != nil {
				return nil, err
			}

			kind := cluster.GetObject().GetObjectKind().GroupVersionKind().Kind
			fleetclusters[fleetmanager.ClusterKey{Kind: kind, Name: cluster.GetObject().GetName()}] = &fleetmanager.FleetCluster{
				Secret:    cluster.GetSecretName(),
				SecretKey: cluster.GetSecretKey(),
				Client:    kclient,
			}
		}
	}
	log.Info("Successful to fetch destination clusters for Rollout")
	return fleetclusters, nil
}

func (a *ApplicationManager) syncRolloutPolicyForCluster(ctx context.Context,
	rolloutPolicy *applicationapi.RolloutConfig,
	destinationClusters map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster,
	policyName string,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	serviceNamespaceName := types.NamespacedName{
		Namespace: rolloutPolicy.Workload.Namespace,
		Name:      rolloutPolicy.ServiceName,
	}

	testloaderNamespaceName := types.NamespacedName{
		Namespace: rolloutPolicy.Workload.Namespace,
		Name:      rolloutPolicy.Workload.Name + "-testloader",
	}

	annotation := map[string]string{
		RolloutIdentifier: policyName,
	}
	provider := rolloutPolicy.TrafficRoutingProvider

	for clusterKey, fleetCluster := range destinationClusters {
		fleetClusterClient := fleetCluster.Client.CtrlRuntimeClient()
		switch provider {
		// If the trafficRoutingProvider is Istio or Kuma, add the sidecar injection label/Annotations to the workload's namespace.
		case fleetapi.Istio, fleetapi.Kuma:
			err := enableSidecarInjection(ctx, fleetClusterClient, rolloutPolicy.Workload.Namespace, provider, annotation)
			if err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "failed to set namespace %s %s's Inject enable", rolloutPolicy.Workload.Namespace, provider)
			}
		case fleetapi.Nginx:
			// Canaries in the same namespace reference the same ingress
			ingress := &ingressv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ingressName,
					Namespace: rolloutPolicy.Workload.Namespace,
				},
			}
			result, err := controllerutil.CreateOrUpdate(ctx, fleetClusterClient, ingress, func() error {
				ingress.SetAnnotations(annotation)
				renderIngress(ingress, rolloutPolicy)
				return nil
			})
			if err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "failed to operate ingress")
			}
			log.Info("sync nginx", "result:", result)

		default:
			return ctrl.Result{}, errors.Errorf("unknown provider type %s", provider)
		}
		log.Info("pre-operation of operating canary success")
		// if delete private testloader when rollout polity has changed
		if rolloutPolicy.TestLoader == nil || !*rolloutPolicy.TestLoader {
			testloaderDeploy := &appsv1.Deployment{}
			if err := deleteResourceCreatedByKurator(ctx, testloaderNamespaceName, fleetClusterClient, testloaderDeploy); err != nil {
				return ctrl.Result{}, err
			}
			testloaderSvc := &corev1.Service{}
			if err := deleteResourceCreatedByKurator(ctx, testloaderNamespaceName, fleetClusterClient, testloaderSvc); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			// Installation of private testloader if needed
			if err := installPrivateTestloader(ctx, testloaderNamespaceName, RolloutIdentifier, policyName, fleetClusterClient); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to install private testloader for workload: %w", err)
			}
		}

		// Get the configuration of the workload's service and generate a canaryService.
		service := &corev1.Service{}
		if err := fleetClusterClient.Get(ctx, serviceNamespaceName, service); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: StatusSyncInterval}, errors.Wrapf(err, "not found service %s in %s", rolloutPolicy.ServiceName, rolloutPolicy.Workload.Namespace)
			}
			return ctrl.Result{}, errors.Wrapf(err, "failed to get service %s in %s", rolloutPolicy.ServiceName, rolloutPolicy.Workload.Namespace)
		}

		canaryInCluster := &flaggerv1b1.Canary{}
		getErr := fleetClusterClient.Get(ctx, serviceNamespaceName, canaryInCluster)
		if getErr != nil && !apierrors.IsNotFound(getErr) {
			return ctrl.Result{}, errors.Wrapf(getErr, "failed to get canary %s in %s", serviceNamespaceName.Name, serviceNamespaceName.Namespace)
		}

		canaryInCluster = renderCanary(*rolloutPolicy, canaryInCluster)
		if canaryService, err := renderCanaryService(*rolloutPolicy, service); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed rander canary configuration")
		} else {
			canaryInCluster.Spec.Service = *canaryService
		}
		if err := applyMetricTemplate(ctx, fleetClusterClient, rolloutPolicy.RolloutPolicy.TrafficAnalysis.Metrics, rolloutPolicy.Workload.Namespace, policyName); err != nil {
			return ctrl.Result{}, err
		}
		canaryInCluster.Spec.Analysis = renderCanaryAnalysis(*rolloutPolicy, clusterKey.Name)
		// Set up annotations to make sure it's a resource created by kurator
		canaryInCluster.SetAnnotations(annotation)

		if apierrors.IsNotFound(getErr) {
			if err := fleetClusterClient.Create(ctx, canaryInCluster); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to create rolloutPolicy: %v", err)
			}
		} else {
			if err := fleetClusterClient.Update(ctx, canaryInCluster); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update rolloutPolicy: %v", err)
			}
		}

		log.Info("sync rolloutPolicy successful")
	}
	return ctrl.Result{}, nil
}

func (a *ApplicationManager) reconcileRolloutSyncStatus(ctx context.Context,
	app *applicationapi.Application,
	fleet *fleetapi.Fleet,
	syncPolicy *applicationapi.ApplicationSyncPolicy,
	policyName string,
) (map[string]*applicationapi.RolloutStatus, error) {
	log := ctrl.LoggerFrom(ctx)

	// depend on fleet and cluster selector get destination clusters
	destinationClusters, err := a.fetchRolloutClusters(ctx, app, a.Client, fleet, syncPolicy)
	if err != nil {
		log.Error(err, "failed to fetch destination clusters for syncPolicy")
		return nil, err
	}

	rolloutStatus := map[string]*applicationapi.RolloutStatus{}
	// Loop all destination cluster to get canary status
	for clusterKey, cluster := range destinationClusters {
		fleetClusterClient := cluster.Client.CtrlRuntimeClient()
		name := generatePolicyResourceName(policyName, clusterKey.Kind, clusterKey.Name)
		canary := &flaggerv1b1.Canary{}
		canaryNamespacedName := types.NamespacedName{
			Namespace: syncPolicy.Rollout.Workload.Namespace,
			Name:      syncPolicy.Rollout.Workload.Name,
		}
		// Use the client of the target cluster to get the status of Flagger canary resources
		if err := fleetClusterClient.Get(ctx, canaryNamespacedName, canary); err != nil {
			return nil, errors.Wrapf(err, "failed to get canary %s in %s", canaryNamespacedName.Name, clusterKey.Name)
		}

		if status, exists := rolloutStatus[name]; exists {
			// If a match is found, update the existing rolloutStatus with the new status.
			status.RolloutStatusInCluster = &canary.Status
		} else {
			currentstatus := applicationapi.RolloutStatus{
				ClusterName:            clusterKey.Name,
				RolloutNameInCluster:   canaryNamespacedName.Name,
				RolloutStatusInCluster: &canary.Status,
			}
			rolloutStatus[name] = &currentstatus
		}
	}

	log.Info("finish get rollout status")
	return rolloutStatus, nil
}

func (a *ApplicationManager) deleteResourcesInMemberClusters(ctx context.Context, app *applicationapi.Application, fleet *fleetapi.Fleet) error {
	log := ctrl.LoggerFrom(ctx)

	for _, syncPolicy := range app.Spec.SyncPolicies {
		rolloutPolicy := syncPolicy.Rollout
		if rolloutPolicy == nil {
			continue
		}
		// Fetch rollout destination clusters. Delete rollout resource in this clusters
		destinationClusters, err := a.fetchRolloutClusters(ctx, app, a.Client, fleet, syncPolicy)
		if err != nil {
			return errors.Wrapf(err, "failed to fetch destination clusters when delete rollout resource")
		}

		namespacedName := types.NamespacedName{
			Namespace: rolloutPolicy.Workload.Namespace,
			Name:      rolloutPolicy.Workload.Namespace,
		}
		serviceNamespaceName := types.NamespacedName{
			Namespace: rolloutPolicy.Workload.Namespace,
			Name:      rolloutPolicy.ServiceName,
		}

		allMetricTemplateNamespaceName := make([]types.NamespacedName, 0, len(rolloutPolicy.RolloutPolicy.TrafficAnalysis.Metrics))
		for _, metric := range rolloutPolicy.RolloutPolicy.TrafficAnalysis.Metrics {
			if metric.CustomMetric != nil {
				metricTemplateNamespaceName := types.NamespacedName{
					Name:      string(metric.Name),
					Namespace: rolloutPolicy.Workload.Namespace,
				}
				allMetricTemplateNamespaceName = append(allMetricTemplateNamespaceName, metricTemplateNamespaceName)
			}
		}

		testloaderNamespaceName := types.NamespacedName{
			Namespace: rolloutPolicy.Workload.Namespace,
			Name:      rolloutPolicy.Workload.Name + "-testloader",
		}
		for _, cluster := range destinationClusters {
			newClient := cluster.Client.CtrlRuntimeClient()

			ns := &corev1.Namespace{}
			if err := deleteResourceCreatedByKurator(ctx, namespacedName, newClient, ns); err != nil {
				return errors.Wrapf(err, "failed to delete namespace")
			}
			if err := deleteIngressCreatedByKurator(ctx, newClient, rolloutPolicy); err != nil {
				return errors.Wrapf(err, "failed to delete ingress")
			}
			testloaderDeploy := &appsv1.Deployment{}
			if err := deleteResourceCreatedByKurator(ctx, testloaderNamespaceName, newClient, testloaderDeploy); err != nil {
				return errors.Wrapf(err, "failed to delete testloader deployment")
			}
			testloaderSvc := &corev1.Service{}
			if err := deleteResourceCreatedByKurator(ctx, testloaderNamespaceName, newClient, testloaderSvc); err != nil {
				return errors.Wrapf(err, "failed to delete testloader service")
			}
			if err := deleteMetricTemplateName(ctx, allMetricTemplateNamespaceName, newClient); err != nil {
				return err
			}
			canary := &flaggerv1b1.Canary{}
			if err := deleteResourceCreatedByKurator(ctx, serviceNamespaceName, newClient, canary); err != nil {
				return errors.Wrapf(err, "failed to delete canary")
			}
		}
	}
	log.Info("delete rollout resource successful")
	return nil
}

func enableSidecarInjection(ctx context.Context, kubeClient client.Client, namespace string, provider fleetapi.Provider, annotation map[string]string) error {
	log := ctrl.LoggerFrom(ctx)

	ns := &corev1.Namespace{}
	namespacedName := types.NamespacedName{
		Namespace: namespace,
		Name:      namespace,
	}
	if err := kubeClient.Get(ctx, namespacedName, ns); err != nil {
		// if no found, create a namespace
		if apierrors.IsNotFound(err) {
			ns.SetName(namespace)
			switch provider {
			case fleetapi.Kuma:
				annotation[kumaInject] = "enabled"
			case fleetapi.Istio:
				ns.SetLabels(map[string]string{istioInject: "enabled"})
			}
			ns.SetAnnotations(annotation)
			if createErr := kubeClient.Create(ctx, ns); createErr != nil {
				return errors.Wrapf(createErr, "failed to create namespace %s", namespacedName.Namespace)
			}
		} else {
			log.Error(err, "failed to get namespace %s", namespacedName.Namespace)
			return err
		}
	} else {
		var newNs client.Object
		switch provider {
		case fleetapi.Kuma:
			newNs = addAnnotations(ns, kumaInject, "enabled")
		case fleetapi.Istio:
			newNs = addLabels(ns, istioInject, "enabled")
		}
		if updateErr := kubeClient.Update(ctx, newNs); updateErr != nil {
			return errors.Wrapf(updateErr, "failed to update namespace %s", namespacedName.Namespace)
		}
	}
	log.Info("Inject sidecar successful")
	return nil
}

func installPrivateTestloader(ctx context.Context, namespacedName types.NamespacedName, annotationKey, annotationValue string, kubeClient client.Client) error {
	log := ctrl.LoggerFrom(ctx)
	// apply testloader deployment resource
	testloaderDeploy, deployErr := render.RenderTestloaderConfig(render.TestlaoderDeployment, namespacedName, annotationKey, annotationValue)
	if deployErr != nil {
		return deployErr
	}
	// b := bytes.NewBuffer(testloaderDeploy)
	deploy := &appsv1.Deployment{}
	if err := yaml.Unmarshal(testloaderDeploy, deploy); err != nil {
		return err
	}

	if createErr := kubeClient.Create(ctx, deploy); createErr != nil {
		if apierrors.IsAlreadyExists(createErr) {
			if updateErr := kubeClient.Update(ctx, deploy); updateErr != nil {
				return errors.Wrapf(updateErr, "failed to update testloader deployment")
			}
		} else {
			return errors.Wrapf(createErr, "failed to create testloader deployment")
		}
	}

	// apply testloader service resource
	testloaderSvc, svcErr := render.RenderTestloaderConfig(render.TestlaoderService, namespacedName, annotationKey, annotationValue)
	if svcErr != nil {
		return svcErr
	}
	svc := &corev1.Service{}
	if err := yaml.Unmarshal(testloaderSvc, svc); err != nil {
		return err
	}

	if createErr := kubeClient.Create(ctx, svc); createErr != nil {
		if apierrors.IsAlreadyExists(createErr) {
			if updateErr := kubeClient.Update(ctx, svc); updateErr != nil {
				return errors.Wrapf(updateErr, "failed to update testloader service")
			}
		} else {
			return errors.Wrapf(createErr, "failed to create testloader service")
		}
	}

	log.Info("install testloader successful")
	return nil
}

func deleteMetricTemplateName(ctx context.Context, allNamespaceName []types.NamespacedName, kubeClient client.Client) error {
	metricTemplate := &flaggerv1b1.MetricTemplate{}
	for _, namespaceName := range allNamespaceName {
		if err := deleteResourceCreatedByKurator(ctx, namespaceName, kubeClient, metricTemplate); err != nil {
			return errors.Wrapf(err, "failed to delete MetricTemplate")
		}
	}
	return nil
}

func deleteResourceCreatedByKurator(ctx context.Context, namespaceName types.NamespacedName, kubeClient client.Client, obj client.Object) error {
	if err := kubeClient.Get(ctx, namespaceName, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to get resource %s in %s", namespaceName.Name, namespaceName.Namespace)
		}
	} else {
		// verify if the deployment were created by kurator
		annotations := obj.GetAnnotations()
		if _, exist := annotations[RolloutIdentifier]; exist {
			if deleteErr := kubeClient.Delete(ctx, obj); deleteErr != nil && !apierrors.IsNotFound(deleteErr) {
				return errors.Wrapf(deleteErr, "failed to delete kubernetes resource")
			}
		}
	}
	return nil
}

func deleteIngressCreatedByKurator(ctx context.Context, kubeClient client.Client, rollout *applicationapi.RolloutConfig) error {
	ingress := &ingressv1.Ingress{}
	namespaceName := types.NamespacedName{
		Namespace: rollout.Workload.Namespace,
		Name:      ingressName,
	}
	if err := kubeClient.Get(ctx, namespaceName, ingress); err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to get ingress %s in %s", namespaceName.Name, namespaceName.Namespace)
		}
	} else {
		// verify if the ingress were created by kurator
		annotations := ingress.GetAnnotations()
		if _, exist := annotations[RolloutIdentifier]; exist {
			labels := ingress.GetLabels()
			if set := sets.New(strings.Split(labels[ingressLabelKey], ",")...); set.Contains(rollout.ServiceName) {
				if set.Len() == 1 {
					if deleteErr := kubeClient.Delete(ctx, ingress); deleteErr != nil && !apierrors.IsNotFound(deleteErr) {
						return errors.Wrapf(deleteErr, "failed to Delete ingress %s in %s", namespaceName.Name, namespaceName.Namespace)
					}
				} else {
					newRules := make([]ingressv1.IngressRule, 0)
					for _, rule := range ingress.Spec.Rules {
						if rule.Host != rollout.RolloutPolicy.TrafficRouting.Host {
							newRules = append(newRules, rule)
						}
					}
					ingress.Spec.Rules = newRules
					labels[ingressLabelKey] = strings.Join(set.Delete(rollout.ServiceName).UnsortedList(), ",")
					ingress.SetLabels(labels)
					if err := kubeClient.Update(ctx, ingress); err != nil {
						return errors.Wrapf(err, "failed to Update ingress %s in %s", namespaceName.Name, namespaceName.Namespace)
					}
				}
			}
		}
	}
	return nil
}

// create/update ingress configuration
func renderIngress(ingress *ingressv1.Ingress, rollout *applicationapi.RolloutConfig) {
	if labels := ingress.GetLabels(); labels == nil || labels[ingressLabelKey] == "" {
		ingress.SetLabels(map[string]string{ingressLabelKey: rollout.ServiceName})
	} else {
		labels[ingressLabelKey] = strings.Join(sets.New(strings.Split(labels[ingressLabelKey], ",")...).Insert(rollout.ServiceName).UnsortedList(), ",")
		ingress.SetLabels(labels)
	}
	addAnnotations(ingress, ingressAnnotationKey, ingressAnnotationValue)
	Prefix := ingressv1.PathTypePrefix
	rule := ingressv1.IngressRule{
		Host: rollout.RolloutPolicy.TrafficRouting.Host,
		IngressRuleValue: ingressv1.IngressRuleValue{
			HTTP: &ingressv1.HTTPIngressRuleValue{
				Paths: []ingressv1.HTTPIngressPath{{
					PathType: &Prefix,
					Path:     "/",
					Backend: ingressv1.IngressBackend{
						Service: &ingressv1.IngressServiceBackend{
							Name: rollout.ServiceName,
							Port: ingressv1.ServiceBackendPort{
								Number: rollout.Port,
							},
						},
					},
				}},
			},
		},
	}
	ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
}

// create/update canary configuration
func renderCanary(rolloutPolicy applicationapi.RolloutConfig, canaryInCluster *flaggerv1b1.Canary) *flaggerv1b1.Canary {
	canaryInCluster.ObjectMeta.Namespace = rolloutPolicy.Workload.Namespace
	canaryInCluster.ObjectMeta.Name = rolloutPolicy.Workload.Name
	canaryInCluster.TypeMeta.Kind = "Canary"
	canaryInCluster.TypeMeta.APIVersion = "flagger.app/v1beta1"
	canaryInCluster.Spec = flaggerv1b1.CanarySpec{
		Provider: string(rolloutPolicy.TrafficRoutingProvider),
		TargetRef: flaggerv1b1.LocalObjectReference{
			APIVersion: rolloutPolicy.Workload.APIVersion,
			Kind:       rolloutPolicy.Workload.Kind,
			Name:       rolloutPolicy.Workload.Name,
		},
		ProgressDeadlineSeconds: rolloutPolicy.RolloutPolicy.RolloutTimeoutSeconds,
		SkipAnalysis:            rolloutPolicy.RolloutPolicy.SkipTrafficAnalysis,
		RevertOnDeletion:        rolloutPolicy.RolloutPolicy.RevertOnDeletion,
		Suspend:                 rolloutPolicy.RolloutPolicy.Suspend,
	}
	switch rolloutPolicy.TrafficRoutingProvider {
	case fleetapi.Nginx:
		canaryInCluster.Spec.IngressRef = &flaggerv1b1.LocalObjectReference{
			APIVersion: ingressAPIVersion,
			Kind:       ingressKind,
			Name:       ingressName,
		}
	case fleetapi.Kuma:
		canaryInCluster.SetAnnotations(map[string]string{"kuma.io/mesh": "default"})
	}
	return canaryInCluster
}

func renderCanaryService(rolloutPolicy applicationapi.RolloutConfig, service *corev1.Service) (*flaggerv1b1.CanaryService, error) {
	if service == nil {
		return nil, fmt.Errorf("service is nil, build canaryService configuration failed")
	}
	ports := service.Spec.Ports
	canaryService := &flaggerv1b1.CanaryService{
		Name: rolloutPolicy.ServiceName,
		Port: rolloutPolicy.Port,
	}
	switch rolloutPolicy.TrafficRoutingProvider {
	case fleetapi.Istio:
		canaryService.Gateways = rolloutPolicy.RolloutPolicy.TrafficRouting.Gateways
		canaryService.Hosts = rolloutPolicy.RolloutPolicy.TrafficRouting.Hosts
		canaryService.Retries = rolloutPolicy.RolloutPolicy.TrafficRouting.Retries
		canaryService.Headers = rolloutPolicy.RolloutPolicy.TrafficRouting.Headers
		canaryService.CorsPolicy = rolloutPolicy.RolloutPolicy.TrafficRouting.CorsPolicy
	case fleetapi.Kuma:
		annotations := &flaggerv1b1.CustomMetadata{Annotations: map[string]string{kumaAnnotation: rolloutPolicy.RolloutPolicy.TrafficRouting.Protocol}}
		canaryService.Apex = annotations
		canaryService.Canary = annotations
		canaryService.Primary = annotations
	}
	Timeout := fmt.Sprintf("%d", rolloutPolicy.RolloutPolicy.TrafficRouting.TimeoutSeconds) + "s"
	canaryService.Timeout = Timeout

	for _, port := range ports {
		if port.Port == rolloutPolicy.Port {
			canaryService.TargetPort = port.TargetPort
			break
		}
	}

	return canaryService, nil
}

func applyMetricTemplate(ctx context.Context, fleetClusterClient client.Client, metrics []applicationapi.Metric, namespace, policyName string) error {
	log := ctrl.LoggerFrom(ctx)
	for _, metric := range metrics {
		if metric.CustomMetric != nil {
			metricTemplate := &flaggerv1b1.MetricTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:        string(metric.Name),
					Namespace:   namespace,
					Annotations: map[string]string{RolloutIdentifier: policyName},
				},
			}
			res, err := controllerutil.CreateOrUpdate(ctx, fleetClusterClient, metricTemplate, func() error {
				metricTemplate.Spec = *metric.CustomMetric
				return nil
			})

			if err != nil {
				return errors.Wrapf(err, "error apply MetricTemplate %s for canary", metric.Name)
			}
			log.Info("success apply", "MetricTemplate:", metric.Name, "result:", res)
		}
	}
	return nil
}

func renderCanaryAnalysis(rolloutPolicy applicationapi.RolloutConfig, clusterName string) *flaggerv1b1.CanaryAnalysis {
	canaryAnalysis := flaggerv1b1.CanaryAnalysis{
		Iterations:      rolloutPolicy.RolloutPolicy.TrafficRouting.AnalysisTimes,
		Threshold:       *rolloutPolicy.RolloutPolicy.TrafficAnalysis.CheckFailedTimes,
		Match:           rolloutPolicy.RolloutPolicy.TrafficRouting.Match,
		SessionAffinity: (*flaggerv1b1.SessionAffinity)(rolloutPolicy.RolloutPolicy.TrafficAnalysis.SessionAffinity),
	}

	if rolloutPolicy.RolloutPolicy.TrafficRouting.CanaryStrategy != nil {
		canaryAnalysis.MaxWeight = rolloutPolicy.RolloutPolicy.TrafficRouting.CanaryStrategy.MaxWeight
		canaryAnalysis.StepWeight = rolloutPolicy.RolloutPolicy.TrafficRouting.CanaryStrategy.StepWeight
		canaryAnalysis.StepWeights = rolloutPolicy.RolloutPolicy.TrafficRouting.CanaryStrategy.StepWeights
		canaryAnalysis.StepWeightPromotion = rolloutPolicy.RolloutPolicy.TrafficRouting.CanaryStrategy.StepWeightPromotion
	}

	CheckInterval := fmt.Sprintf("%d", *rolloutPolicy.RolloutPolicy.TrafficAnalysis.CheckIntervalSeconds) + "s"
	canaryAnalysis.Interval = CheckInterval

	canaryMetric := []flaggerv1b1.CanaryMetric{}
	for _, metric := range rolloutPolicy.RolloutPolicy.TrafficAnalysis.Metrics {
		metricInterval := fmt.Sprintf("%d", *metric.IntervalSeconds) + "s"
		templateMetric := flaggerv1b1.CanaryMetric{
			Name:           string(metric.Name),
			Interval:       metricInterval,
			ThresholdRange: (*flaggerv1b1.CanaryThresholdRange)(metric.ThresholdRange),
		}
		if metric.Name != applicationapi.RequestSuccessRate && metric.Name != applicationapi.RequestDuration {
			templateMetric.TemplateRef = &flaggerv1b1.CrossNamespaceObjectReference{
				Name:      string(metric.Name),
				Namespace: rolloutPolicy.Workload.Namespace,
			}
		}
		canaryMetric = append(canaryMetric, templateMetric)
	}
	canaryAnalysis.Metrics = canaryMetric

	// Trigger testloader to request service before analysis by webhook.
	webhookTemplate := flaggerv1b1.CanaryWebhook{
		Name:    "generated-testload",
		Timeout: "60s",
	}

	if len(rolloutPolicy.RolloutPolicy.TrafficAnalysis.Webhooks.Commands) != 0 {
		var url string
		// if have private webhook, webhook url is private testloader url
		// else is public testloader url
		if rolloutPolicy.TestLoader != nil && *rolloutPolicy.TestLoader {
			name := rolloutPolicy.ServiceName + "-testloader"
			namespace := rolloutPolicy.Workload.Namespace
			url = generateWebhookUrl(name, namespace)
		} else if namespace, exist := plugin.ProviderNamespace[fleetapi.Provider(rolloutPolicy.TrafficRoutingProvider)]; exist {
			name := namespace + "-testloader-" + clusterName + "-loadtester"
			url = generateWebhookUrl(name, namespace)
		}
		webhookTemplate.URL = url

		timeout := fmt.Sprintf("%d", *rolloutPolicy.RolloutPolicy.TrafficAnalysis.Webhooks.TimeoutSeconds) + "s"
		webhookTemplate.Timeout = timeout

		canaryWebhook := []flaggerv1b1.CanaryWebhook{}
		bakName := webhookTemplate.Name
		for index, command := range rolloutPolicy.RolloutPolicy.TrafficAnalysis.Webhooks.Commands {
			metadata := map[string]string{
				"type": "cmd",
				"cmd":  command,
			}
			webhookTemplate.Metadata = &metadata
			webhookTemplate.Name = bakName + "-" + fmt.Sprintf("%d", index)
			canaryWebhook = append(canaryWebhook, webhookTemplate)
		}

		canaryAnalysis.Webhooks = canaryWebhook
	}
	return &canaryAnalysis
}

func generateWebhookUrl(name, namespace string) string {
	url := "http://" + name + "." + namespace + "/"
	return url
}

func addLabels(obj client.Object, key, value string) client.Object {
	labels := obj.GetLabels()
	// prevent nil pointer panic
	if labels == nil {
		obj.SetLabels(map[string]string{
			key: value,
		})
		return obj
	}
	labels[key] = value
	obj.SetLabels(labels)
	return obj
}

func addAnnotations(obj client.Object, keysAndValues ...string) client.Object {
	annotations := obj.GetAnnotations()
	// prevent nil pointer panic
	if annotations == nil {
		annotations = make(map[string]string)
	}
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			annotations[keysAndValues[i]] = keysAndValues[i+1]
		} else {
			annotations[keysAndValues[i]] = ""
		}
	}
	obj.SetAnnotations(annotations)
	return obj
}

func mergeMap(map1, map2 map[string]*applicationapi.RolloutStatus) map[string]*applicationapi.RolloutStatus {
	for name, rolloutStatus := range map1 {
		if _, exist := map2[name]; !exist {
			map2[name] = rolloutStatus
		}
	}
	return map2
}
