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

package prometheus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"

	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/resource"
	"sigs.k8s.io/yaml"

	"kurator.dev/kurator/manifests"
	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/typemeta"
	"kurator.dev/kurator/pkg/util"
)

const (
	promELBSvc                 = "prometheus-elb"
	additinalScrapeConfigsName = "additional-scrape-configs"
	additinalScrapeConfigsKey  = "prometheus-additional.yaml"
	prometheusOperatorName     = "prometheus-operator"
	promCRDName                = "prometheuses.monitoring.coreos.com"
	monitoringNamespace        = "monitoring"
	promSvcName                = "prometheus-k8s"

	setupDir   = "profiles/prom/setup"
	promDir    = "profiles/prom"
	promCRFile = promDir + "/prometheus-prometheus.yaml"
)

var prometheusGVK = schema.GroupVersionKind{
	Group:   "monitoring.coreos.com",
	Version: "v1",
	Kind:    "Prometheus",
}

type InstallArgs struct {
	Primary string
}

type Plugin struct {
	*client.Client

	args    *InstallArgs
	options *generic.Options

	primary  string
	clusters []string
}

func NewPlugin(s *generic.Options, args *InstallArgs) (*Plugin, error) {
	plugin := &Plugin{
		options: s,
		args:    args,
	}
	rest := s.RESTClientGetter()
	c, err := client.NewClient(rest)
	if err != nil {
		return nil, err
	}
	plugin.Client = c

	return plugin, nil
}

func (p *Plugin) Execute(cmdArgs, environment []string) error {
	if err := p.preCheck(); err != nil {
		return err
	}

	if err := p.runInstall(); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) preCheck() error {
	clusters, err := p.getClusters()
	if err != nil {
		return err
	}

	p.clusters = clusters
	p.primary = p.args.Primary

	return nil
}

func (p *Plugin) getClusters() ([]string, error) {
	clusters, err := p.KarmadaClient().ClusterV1alpha1().Clusters().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	clusterNames := make([]string, 0, len(clusters.Items))
	for _, c := range clusters.Items {
		clusterNames = append(clusterNames, c.Name)
	}

	return clusterNames, nil
}

// runInstall create prometheus operator
// 1. create all resources in profiles/prom/step(most are CRDs)
// 2. create other related resources in profiles/prom/*.yaml(non-recursive)
// 3. apply ClusterPropagationPolicy and PropagationPolicy
func (p *Plugin) runInstall() error {
	// create namespaces and crds first
	setupResourceList, err := p.resources(setupDir)
	if err != nil {
		return fmt.Errorf("load resource fail, %w", err)
	}
	if _, err := p.HelmClient().Update(setupResourceList, setupResourceList, true); err != nil {
		return fmt.Errorf("create setup resource fail, %w", err)
	}

	if err := util.WaitCRDReady(p.CrdClient(), promCRDName, p.options.WaitInterval, p.options.WaitTimeout); err != nil {
		return fmt.Errorf("wait CRD %s ready fail, %w", promCRDName, err)
	}

	logrus.Infof("CRD is ready, start to install prometheus")
	resourceList, err := p.resources(promDir)
	if err != nil {
		return fmt.Errorf("load resource fail, %w", err)
	}
	if _, err := p.HelmClient().Update(resourceList, resourceList, true); err != nil {
		return fmt.Errorf("create resource fail, %w", err)
	}

	if err := p.exposePrometheus(); err != nil {
		return err
	}

	// creating OverridePolicy first if enable federation
	if p.useFederation() {
		logrus.Debugf("create prometheus additional scrape config")
		if err := p.createAdditionalScrapeConfigs(); err != nil {
			return err
		}
	}

	// merge all resources before generate policy
	for _, r := range setupResourceList {
		resourceList.Append(r)
	}
	cpp, pp := p.generatePolicy(resourceList)

	if err := p.UpdateResource(cpp); err != nil {
		return fmt.Errorf("apply ClusterPropagationPolicy fail, %v", err)
	}

	// wait apiEnablement in allCluster
	if err := util.WaitAPIEnableInClusters(p.KarmadaClient(), prometheusGVK, p.clusters, p.options.WaitInterval, p.options.WaitTimeout); err != nil {
		return fmt.Errorf("wait CRD %s ready fail, %w", promCRDName, err)
	}
	logrus.Debugf("prometheus API enabled in all clusters")

	if err := p.UpdateResource(pp); err != nil {
		return fmt.Errorf("apply PropagationPolicy fail, %v", err)
	}

	return nil
}

func (p *Plugin) exposePrometheus() error {
	s, err := p.KubeClient().CoreV1().Services(monitoringNamespace).Get(context.TODO(), promSvcName, metav1.GetOptions{})
	if err != nil {
		return nil
	}

	// 2. create elb service for prometheus
	promElbSvc := &corev1.Service{
		TypeMeta: typemeta.Service,
		ObjectMeta: metav1.ObjectMeta{
			Name:      promELBSvc,
			Namespace: monitoringNamespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Protocol: corev1.ProtocolTCP,
					Port:     9090,
				},
			},
			Selector:        s.Spec.Selector,
			SessionAffinity: corev1.ServiceAffinityNone,
			Type:            corev1.ServiceTypeLoadBalancer,
		},
	}
	if err := p.UpdateResource(promElbSvc); err != nil {
		return err
	}

	pp := &policyv1alpha1.PropagationPolicy{
		TypeMeta: typemeta.PropagationPolicy,
		ObjectMeta: metav1.ObjectMeta{
			Name:      promElbSvc.Name,
			Namespace: promElbSvc.Namespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: "v1",
					Kind:       "Service",
					Name:       promElbSvc.Name,
					Namespace:  promElbSvc.Namespace,
				},
			},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: p.clusters,
				},
			},
		},
	}

	if err := p.UpdateResource(pp); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) getFederalEndpoints() ([]endpoint, error) {
	endpoints := make([]endpoint, 0)
	err := wait.PollImmediate(p.options.WaitInterval, p.options.WaitTimeout, func() (done bool, err error) {
		svc, err := p.KubeClient().CoreV1().Services(monitoringNamespace).Get(context.TODO(), promELBSvc, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		clusterNames := sets.NewString(p.clusters...)
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if ingress.Hostname == p.primary {
				continue
			}

			// kamada make sure `ingress.Hostname` exist
			if clusterNames.Has(ingress.Hostname) {
				endpoints = append(endpoints, endpoint{
					name:    ingress.Hostname,
					address: ingress.IP,
				})
			}
		}

		return len(endpoints) >= len(clusterNames)-1, nil
	})

	return endpoints, err
}

func (p *Plugin) createAdditionalScrapeConfigs() error {
	endpoints, err := p.getFederalEndpoints()
	if err != nil {
		return err
	}

	cfg, err := genAdditionalScrapeConfigs(endpoints)
	if err != nil {
		return err
	}
	// create additional scrape configuration
	additionalScrapeConfigs := &corev1.Secret{
		TypeMeta: typemeta.Secret,
		ObjectMeta: metav1.ObjectMeta{
			Name:      additinalScrapeConfigsName,
			Namespace: monitoringNamespace,
		},
		StringData: map[string]string{
			additinalScrapeConfigsKey: cfg,
		},
	}
	if err := p.UpdateResource(additionalScrapeConfigs); err != nil {
		return err
	}

	val, err := json.Marshal(&corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: additinalScrapeConfigsName,
		},
		Key: additinalScrapeConfigsKey,
	})
	if err != nil {
		return err
	}

	promCfg, err := p.loadProm()
	if err != nil {
		return err
	}
	if promCfg.Namespace == "" {
		return fmt.Errorf("get prom faild, %+v", promCfg)
	}
	op := &policyv1alpha1.OverridePolicy{
		TypeMeta: typemeta.OverridePolicy,
		ObjectMeta: metav1.ObjectMeta{
			Name:      promCfg.Name,
			Namespace: promCfg.Namespace,
		},
		Spec: policyv1alpha1.OverrideSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: promCfg.APIVersion,
					Kind:       promCfg.Kind,
					Name:       promCfg.Name,
					Namespace:  promCfg.Namespace,
				},
			},
			OverrideRules: []policyv1alpha1.RuleWithCluster{
				{
					Overriders: policyv1alpha1.Overriders{
						Plaintext: []policyv1alpha1.PlaintextOverrider{
							{
								Path:     "/spec/additionalScrapeConfigs",
								Operator: policyv1alpha1.OverriderOpAdd,
								Value:    apiextv1.JSON{Raw: val},
							},
						},
					},
					TargetCluster: &policyv1alpha1.ClusterAffinity{
						ClusterNames: []string{p.primary},
					},
				},
			},
		},
	}
	if err := p.UpdateResource(op); err != nil {
		return fmt.Errorf("create OverridePolicy fail, %w", err)
	}

	pp := &policyv1alpha1.PropagationPolicy{
		TypeMeta: typemeta.PropagationPolicy,
		ObjectMeta: metav1.ObjectMeta{
			Name:      additionalScrapeConfigs.Name,
			Namespace: additionalScrapeConfigs.Namespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: additionalScrapeConfigs.APIVersion,
					Kind:       additionalScrapeConfigs.Kind,
					Name:       additionalScrapeConfigs.Name,
					Namespace:  additionalScrapeConfigs.Namespace,
				},
			},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: []string{p.primary},
				},
			},
		},
	}
	if err := p.UpdateResource(pp); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) generatePolicy(resourceList kube.ResourceList) (
	*policyv1alpha1.ClusterPropagationPolicy,
	*policyv1alpha1.PropagationPolicy) {
	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		TypeMeta: typemeta.ClusterPropagationPolicy,
		ObjectMeta: metav1.ObjectMeta{
			Name: prometheusOperatorName,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: p.clusters,
				},
			},
		},
	}

	pp := &policyv1alpha1.PropagationPolicy{
		TypeMeta: typemeta.PropagationPolicy,
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusOperatorName,
			Namespace: monitoringNamespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: p.clusters,
				},
			},
		},
	}

	util.AppendResourceSelector(cpp, pp, resourceList)

	return cpp, pp
}

func (p *Plugin) resources(root string) (kube.ResourceList, error) {
	fsys := manifests.BuiltinOrDir("")
	files, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, err
	}

	infos := []*resource.Info{}
	for _, fname := range files {
		if fname.IsDir() {
			continue
		}

		f := path.Join(root, fname.Name())
		b, err := fs.ReadFile(fsys, f)
		if err != nil {
			// should not happen
			continue
		}

		resourceList, err := p.HelmClient().Build(bytes.NewBuffer(b), false)
		if err != nil {
			// should not happen
			continue
		}

		infos = append(infos, resourceList...)
	}

	return infos, nil
}

func (p *Plugin) loadProm() (*monitoringv1.Prometheus, error) {
	fsys := manifests.BuiltinOrDir("")
	f, err := fs.ReadFile(fsys, promCRFile)
	if err != nil {
		return nil, err
	}
	prom := &monitoringv1.Prometheus{}
	if err := yaml.Unmarshal(f, prom); err != nil {
		return nil, err
	}

	return prom, nil
}

func (p *Plugin) useFederation() bool {
	return p.primary != ""
}
