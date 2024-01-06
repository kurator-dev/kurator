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
	"io/fs"
	"time"

	flaggerv1b1 "github.com/fluxcd/flagger/pkg/apis/flagger/v1beta1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	applicationapi "kurator.dev/kurator/pkg/apis/apps/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
	fleetmanager "kurator.dev/kurator/pkg/fleet-manager"
	"kurator.dev/kurator/pkg/fleet-manager/manifests"
	plugin "kurator.dev/kurator/pkg/fleet-manager/plugin"
)

const (
	// kurator rollout labels
	RolloutLabel  = "kurator.dev/rollout"
	sidecarInject = "istio-injection"

	// testloader configuration path
	testloaderDeployPath = "plugins/testloader-deploy.yaml"
	testloaderSvcPath    = "plugins/testloader-svc.yaml"
	// StatusSyncInterval specifies the interval for requeueing when synchronizing status. It determines how frequently the status should be checked and updated.
	StatusSyncInterval = 30 * time.Second
)

func (a *ApplicationManager) fetchRolloutClusters(ctx context.Context,
	app *applicationapi.Application,
	kubeClient client.Client,
	fleet *fleetapi.Fleet,
	syncPolicy *applicationapi.ApplicationSyncPolicy,
) (map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster, error) {
	log := ctrl.LoggerFrom(ctx)
	destination := getPolicyDestination(app, syncPolicy)
	ClusterInterfaceList, result, err := a.fetchFleetClusterList(ctx, fleet, destination.ClusterSelector)
	if err != nil || result.RequeueAfter > 0 {
		return nil, err
	}

	fleetclusters := make(map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster, len(ClusterInterfaceList))
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
	log.Info("Successful to fetch destination clusters for Rollout")
	return fleetclusters, nil
}

func (a *ApplicationManager) syncRolloutPolicyForCluster(ctx context.Context,
	rolloutPolicy *applicationapi.RolloutConfig,
	destinationClusters map[fleetmanager.ClusterKey]*fleetmanager.FleetCluster,
	policyName string,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	annotations := map[string]string{
		RolloutLabel: policyName,
	}

	serviceNamespaceName := types.NamespacedName{
		Namespace: rolloutPolicy.Workload.Namespace,
		Name:      rolloutPolicy.ServiceName,
	}

	testloaderNamespaceName := types.NamespacedName{
		Namespace: rolloutPolicy.Workload.Namespace,
		Name:      rolloutPolicy.Workload.Name + "-testloader",
	}

	for clusterKey, fleetCluster := range destinationClusters {
		newClient := fleetCluster.Client.CtrlRuntimeClient()

		// if trafficRoutingProvider is istio, find workload namespace with Istio sidecar injection enabled.
		if rolloutPolicy.TrafficRoutingProvider == "istio" {
			err := namespaceSidecarInject(ctx, newClient, rolloutPolicy.Workload.Namespace)
			if err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "failed to set namespace %s istio-injection enable", rolloutPolicy.Workload.Namespace)
			}
		}

		// if delete private testloader when rollout polity has changed
		if rolloutPolicy.TestLoader == nil || !*rolloutPolicy.TestLoader {
			testloaderDeploy := &appsv1.Deployment{}
			if err := deleteResourceCreateByKurator(ctx, testloaderNamespaceName, newClient, testloaderDeploy); err != nil {
				return ctrl.Result{}, err
			}
			testloaderSvc := &corev1.Service{}
			if err := deleteResourceCreateByKurator(ctx, testloaderNamespaceName, newClient, testloaderSvc); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Installation of private testloader if needed
		if rolloutPolicy.TestLoader != nil && *rolloutPolicy.TestLoader {
			if result, err := installPrivateTestloader(ctx, testloaderNamespaceName, *fleetCluster, annotations); err != nil {
				return result, fmt.Errorf("failed to install private testloader for workload: %w", err)
			}
		}

		// Get the configuration of the workload's service and generate a canaryService.
		service := &corev1.Service{}
		if err := newClient.Get(ctx, serviceNamespaceName, service); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: StatusSyncInterval}, errors.Wrapf(err, "not found service %s in %s", rolloutPolicy.ServiceName, rolloutPolicy.Workload.Namespace)
			}
			return ctrl.Result{}, errors.Wrapf(err, "failed to get service %s in %s", rolloutPolicy.ServiceName, rolloutPolicy.Workload.Namespace)
		}

		canaryInCluster := &flaggerv1b1.Canary{}
		getErr := newClient.Get(ctx, serviceNamespaceName, canaryInCluster)
		canaryInCluster = renderCanary(*rolloutPolicy, canaryInCluster)
		if canaryService, err := renderCanaryService(*rolloutPolicy, service); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed rander canary configuration")
		} else {
			canaryInCluster.Spec.Service = *canaryService
		}
		canaryInCluster.Spec.Analysis = renderCanaryAnalysis(*rolloutPolicy, clusterKey.Name)
		// Set up annotations to make sure it's a resource created by kurator
		canaryInCluster.SetAnnotations(annotations)

		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				if err := newClient.Create(ctx, canaryInCluster); err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to create rolloutPolicy: %v", err)
				}
			}
			return ctrl.Result{}, errors.Wrapf(getErr, "failed to get canary %s in %s", serviceNamespaceName.Name, serviceNamespaceName.Namespace)
		}
		if err := newClient.Update(ctx, canaryInCluster); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update rolloutPolicy: %v", err)
		}

		log.Info("sync rolloutPolicy for cluster successful")
	}
	return ctrl.Result{}, nil
}

func namespaceSidecarInject(ctx context.Context, kubeClient client.Client, namespace string) error {
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
			ns := addLablesOrAnnotaions(ns, "labels", sidecarInject, "enabled")
			if createErr := kubeClient.Create(ctx, ns); createErr != nil {
				return errors.Wrapf(createErr, "failed to create namespace %s", namespacedName.Namespace)
			}
		}
		ns := addLablesOrAnnotaions(ns, "labels", sidecarInject, "enabled")
		if updateErr := kubeClient.Update(ctx, ns); updateErr != nil {
			return errors.Wrapf(updateErr, "failed to update namespace %s", namespacedName.Namespace)
		}
	}
	log.Info("Inject sidecar successful")
	return nil
}

func installPrivateTestloader(ctx context.Context,
	namespacedName types.NamespacedName,
	fleetCluster fleetmanager.FleetCluster,
	annotations map[string]string,
) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	clusterClient := fleetCluster.Client.CtrlRuntimeClient()

	// Creating a private testload deployment from a configuration file
	filepath := manifests.BuiltinOrDir("")
	deploy, err1 := generateDeployConfig(filepath, testloaderDeployPath, namespacedName.Name, namespacedName.Namespace)
	if err1 != nil {
		return ctrl.Result{}, fmt.Errorf("failed get testloader deployment configuration: %v", err1)
	}
	// Set up annotations to make sure it's a resource created by kurator.
	deploy.SetAnnotations(annotations)

	if err := clusterClient.Create(ctx, deploy); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if updateErr := clusterClient.Update(ctx, deploy); updateErr != nil {
				return ctrl.Result{}, errors.Wrapf(err, "failed to update private testloader deployment")
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrapf(err, "failed to create private testloader deployment")
	}

	// Creating a private testload service from a configuration file
	svc, err2 := generateSvcConfig(filepath, testloaderSvcPath, namespacedName.Name, namespacedName.Namespace)
	if err2 != nil {
		return ctrl.Result{}, fmt.Errorf("failed get testloader service configuration: %v", err2)
	}
	// Set up annotations to make sure it's a resource created by kurator.
	svc.SetAnnotations(annotations)

	if err := clusterClient.Create(ctx, svc); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if updateErr := clusterClient.Update(ctx, svc); updateErr != nil {
				return ctrl.Result{}, errors.Wrapf(err, "failed to update private testloader service")
			}
		}
		return ctrl.Result{}, errors.Wrapf(err, "failed to create private testloader service")
	}

	log.Info("Create private workload successful")
	return ctrl.Result{}, nil
}

func generateDeployConfig(fsys fs.FS, fileName, name, namespace string) (*appsv1.Deployment, error) {
	file, err := fs.ReadFile(fsys, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open telstloader deployment configuration: %v", err)
	}

	deploy := appsv1.Deployment{}
	if err := yaml.Unmarshal(file, &deploy); err != nil {
		return nil, err
	}

	deploy.SetName(name)
	deploy.SetNamespace(namespace)
	// Set up labels to make sure it's a resource created by kurator.
	deploy.SetLabels(map[string]string{
		"app": name,
	})
	// let svc's selector to select private testloader pod
	deploy.Spec.Selector.MatchLabels = map[string]string{
		"app": name,
	}
	deploy.Spec.Template.ObjectMeta.Labels = map[string]string{
		"app": name,
	}

	return &deploy, nil
}

func generateSvcConfig(fsys fs.FS, fileName string, name, namespace string) (*corev1.Service, error) {
	file, err1 := fs.ReadFile(fsys, fileName)
	if err1 != nil {
		return nil, fmt.Errorf("failed to open telstloader service configuration: %v", err1)
	}

	svc := corev1.Service{}
	if err := yaml.Unmarshal(file, &svc); err != nil {
		return nil, err
	}

	svc.SetName(name)
	svc.SetNamespace(namespace)
	// Set up labels to make sure it's a resource created by kurator.
	svc.SetLabels(map[string]string{
		"app": name,
	})
	// let svc's selector to select private testloader pod
	svc.Spec.Selector = map[string]string{
		"app": name,
	}

	return &svc, nil
}

// create/update canary configuration
func renderCanary(rolloutPolicy applicationapi.RolloutConfig, canaryInCluster *flaggerv1b1.Canary) *flaggerv1b1.Canary {
	value := int32(*rolloutPolicy.RolloutPolicy.RolloutTimeoutSeconds)
	ptrValue := &value

	canaryInCluster.ObjectMeta.Namespace = rolloutPolicy.Workload.Namespace
	canaryInCluster.ObjectMeta.Name = rolloutPolicy.Workload.Name
	canaryInCluster.TypeMeta.Kind = "Canary"
	canaryInCluster.TypeMeta.APIVersion = "flagger.app/v1beta1"
	canaryInCluster.Spec = flaggerv1b1.CanarySpec{
		Provider: rolloutPolicy.TrafficRoutingProvider,
		TargetRef: flaggerv1b1.LocalObjectReference{
			APIVersion: rolloutPolicy.Workload.APIVersion,
			Kind:       rolloutPolicy.Workload.Kind,
			Name:       rolloutPolicy.Workload.Name,
		},
		ProgressDeadlineSeconds: ptrValue,
		SkipAnalysis:            rolloutPolicy.RolloutPolicy.SkipTrafficAnalysis,
		RevertOnDeletion:        rolloutPolicy.RolloutPolicy.RevertOnDeletion,
		Suspend:                 rolloutPolicy.RolloutPolicy.Suspend,
	}

	return canaryInCluster
}

func renderCanaryService(rolloutPolicy applicationapi.RolloutConfig, service *corev1.Service) (*flaggerv1b1.CanaryService, error) {
	if service == nil {
		return nil, fmt.Errorf("service is nil, build canaryService configuration failed")
	}
	ports := service.Spec.Ports
	canaryService := &flaggerv1b1.CanaryService{
		Name:       rolloutPolicy.ServiceName,
		Port:       rolloutPolicy.Port,
		Gateways:   rolloutPolicy.RolloutPolicy.TrafficRouting.Gateways,
		Hosts:      rolloutPolicy.RolloutPolicy.TrafficRouting.Hosts,
		Retries:    rolloutPolicy.RolloutPolicy.TrafficRouting.Retries,
		Headers:    rolloutPolicy.RolloutPolicy.TrafficRouting.Headers,
		CorsPolicy: rolloutPolicy.RolloutPolicy.TrafficRouting.CorsPolicy,
		Primary:    (*flaggerv1b1.CustomMetadata)(rolloutPolicy.Primary),
		Canary:     (*flaggerv1b1.CustomMetadata)(rolloutPolicy.Preview),
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

func deleteResourceCreateByKurator(ctx context.Context, namespaceName types.NamespacedName, kubeClient client.Client, obj client.Object) error {
	if err := kubeClient.Get(ctx, namespaceName, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrapf(err, "get kubernetes resource error")
		}
	} else {
		// verify if the deployment were created by kurator
		annotations := obj.GetAnnotations()
		if _, exist := annotations[RolloutLabel]; exist {
			if deleteErr := kubeClient.Delete(ctx, obj); deleteErr != nil {
				return errors.Wrapf(deleteErr, "failed to delete kubernetes resource")
			}
		}
	}
	return nil
}

func addLablesOrAnnotaions(obj client.Object, labelsOrAnnotaions, key, value string) client.Object {
	switch labelsOrAnnotaions {
	case "labels":
		labels := obj.GetLabels()
		if labels == nil {
			obj.SetLabels(map[string]string{
				key: value,
			})
			return obj
		}
		labels[key] = value
		obj.SetLabels(labels)
	case "annotations":
		annotations := obj.GetAnnotations()
		if annotations == nil {
			obj.SetAnnotations(map[string]string{
				key: value,
			})
			return obj
		}
		annotations[key] = value
		obj.SetAnnotations(annotations)
	}

	return obj
}
