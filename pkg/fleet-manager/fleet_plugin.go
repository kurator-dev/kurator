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
	"sync"

	hrapiv2b1 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"helm.sh/helm/v3/pkg/kube"
	"istio.io/istio/pkg/util/sets"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

const (
	MonitoringNamespace         = "monitoring"
	PrometheusThanosServiceName = "prometheus-prometheus-thanos"

	NoneClusterIP = "None"
)

func (f *FleetManager) reconcilePlugins(ctx context.Context, fleet *fleetapi.Fleet, fleetClusters map[ClusterKey]*fleetCluster) (ctrl.Result, error) {
	var resources kube.ResourceList

	if fleet.Spec.Plugin != nil {
		type reconcileResult struct {
			result     kube.ResourceList
			ctrlResult ctrl.Result
			err        error
		}

		type reconcileFunc func(context.Context, *fleetapi.Fleet, map[ClusterKey]*fleetCluster) (kube.ResourceList, ctrl.Result, error)

		funcs := []reconcileFunc{
			f.reconcileMetricPlugin,
			f.reconcileGrafanaPlugin,
			f.reconcileKyvernoPlugin,
			f.reconcileBackupPlugin,
		}

		resultsChannel := make(chan reconcileResult, len(funcs))
		var wg sync.WaitGroup

		for _, fn := range funcs {
			wg.Add(1)
			go func(fn reconcileFunc) {
				defer wg.Done()
				result, ctrlResult, err := fn(ctx, fleet, fleetClusters)
				resultsChannel <- reconcileResult{result, ctrlResult, err}
			}(fn)
		}

		go func() {
			wg.Wait()
			close(resultsChannel)
		}()

		var errs []error
		var ctrlResults []ctrl.Result

		for res := range resultsChannel {
			if res.err != nil {
				errs = append(errs, res.err)
			}
			if res.ctrlResult.Requeue || res.ctrlResult.RequeueAfter > 0 {
				ctrlResults = append(ctrlResults, res.ctrlResult)
			}
			resources = append(resources, res.result...)
		}

		if len(errs) > 0 {
			// Combine all errors into one error message
			return ctrl.Result{}, fmt.Errorf("encountered multiple errors: %v", errs)
		}

		if len(ctrlResults) > 0 {
			// Handle multiple ctrlResults
			return ctrlResults[0], nil // TODO: if we need strategy about RequeueAfter
		}
	}

	return f.reconcilePluginResources(ctx, fleet, resources)
}

// reconcilePluginResources delete redundant HelmRelease and HelmRepository resources,
// for example, disable metric plugin will try to delete metric plugin resources.
func (f *FleetManager) reconcilePluginResources(ctx context.Context, fleet *fleetapi.Fleet, resources kube.ResourceList) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log = log.WithValues("fleet", types.NamespacedName{Name: fleet.Name, Namespace: fleet.Namespace})

	repoDict, releaseDict := sets.New[types.NamespacedName](), sets.New[types.NamespacedName]()
	for _, res := range resources {
		switch res.Mapping.GroupVersionKind.Kind {
		case sourcev1beta2.HelmRepositoryKind:
			repoDict.Insert(types.NamespacedName{Namespace: res.Namespace, Name: res.Name})
		case hrapiv2b1.HelmReleaseKind:
			releaseDict.Insert(types.NamespacedName{Namespace: res.Namespace, Name: res.Name})
		default:
			// should not happen, but just in case
			log.V(2).Info("unexpected resource type", "kind", res.Mapping.GroupVersionKind.Kind)
		}
	}

	helmRepos := &sourcev1beta2.HelmRepositoryList{}
	helmReleases := &hrapiv2b1.HelmReleaseList{}
	fleetLabels := fleetResourceLabels(fleet.Name)
	if err := f.Client.List(ctx, helmRepos, client.InNamespace(fleet.Namespace), fleetLabels); err != nil {
		log.Error(err, "failed to list helm repository")
		return ctrl.Result{}, err
	}
	if err := f.Client.List(ctx, helmReleases, client.InNamespace(fleet.Namespace), fleetLabels); err != nil {
		log.Error(err, "failed to list helm release")
		return ctrl.Result{}, err
	}

	for _, repo := range helmRepos.Items {
		if !repoDict.Contains(types.NamespacedName{Namespace: repo.Namespace, Name: repo.Name}) {
			// delete redundant helm releases
			if err := f.Client.Delete(ctx, &repo); err != nil {
				log.Error(err, "failed to delete helm repository")
				return ctrl.Result{}, err
			}
		}
	}

	for _, release := range helmReleases.Items {
		// delete redundant helm releases
		if !releaseDict.Contains(types.NamespacedName{Namespace: release.Namespace, Name: release.Name}) {
			if err := f.Client.Delete(ctx, &release); err != nil {
				log.Error(err, "failed to delete helm release")
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (f *FleetManager) helmReleaseReady(ctx context.Context, fleet *fleetapi.Fleet, resources kube.ResourceList) bool {
	log := ctrl.LoggerFrom(ctx)
	log = log.WithValues("fleet", types.NamespacedName{Name: fleet.Name, Namespace: fleet.Namespace})

	for _, res := range resources {
		switch res.Mapping.GroupVersionKind.Kind {
		case hrapiv2b1.HelmReleaseKind:
			// Wait for all helm releases to be reconciled
			hr := &hrapiv2b1.HelmRelease{}
			if err := f.Client.Get(ctx, types.NamespacedName{
				Namespace: fleet.Namespace,
				Name:      res.Name,
			}, hr); err != nil {
				return false
			}

			if !isReady(hr.Status.Conditions) {
				log.Info("helm release is not ready", "helm release", hr.Name)
				return false
			}
		default:
			continue
		}
	}

	return true
}
