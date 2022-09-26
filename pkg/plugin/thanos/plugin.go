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

package thanos

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"

	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	karmadautil "github.com/karmada-io/karmada/pkg/util"
	"github.com/sirupsen/logrus"
	helmclient "helm.sh/helm/v3/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/cli-runtime/pkg/resource"

	"kurator.dev/kurator/manifests"
	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/util"
)

const (
	thanosNamespace         = "thanos"
	thanosPolicyName        = "thanos"
	promCRDName             = "prometheuses.monitoring.coreos.com"
	prometheusCRName        = "thanos"
	monitoringNamespace     = "monitoring"
	thanosSidecarELBSvcName = "thanos-sidecar-elb"
	thanosQuerySvcName      = "thanos-query"

	setupDir  = "profiles/prom-thanos/setup"
	promDir   = "profiles/prom-thanos"
	thanosDir = "profiles/thanos"
)

var (
	prometheusGVK = schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "Prometheus",
	}
)

type InstallArgs struct {
	HostKubeconfig string
	HostContext    string

	ObjectStoreConfig string
}

type Plugin struct {
	*client.Client

	args    *InstallArgs
	options *generic.Options

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

	if err := p.runInstallPrometheus(); err != nil {
		return err
	}

	if err := p.runInstallThanos(thanosNamespace); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) preCheck() error {
	if _, err := os.Stat(p.args.ObjectStoreConfig); err != nil {
		return fmt.Errorf("check object store config fail, %w", err)
	}

	clusters, err := p.getClusters()
	if err != nil {
		return err
	}

	p.clusters = clusters

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
// 1. create all resources in profiles/prom-thanos/step(most are CRDs)
// 2. create other related resources in profiles/prom-thanos/*.yaml(non-recursive)
// 3. apply ClusterPropagationPolicy and PropagationPolicy
func (p *Plugin) runInstallPrometheus() error {
	// create namespaces and crds first
	helmClient := p.HelmClient()
	setupResourceList, err := p.resources(helmClient, setupDir)
	if err != nil {
		return fmt.Errorf("load resource fail, %w", err)
	}

	if _, err := helmClient.Update(setupResourceList, setupResourceList, true); err != nil {
		return fmt.Errorf("create setup resource fail, %w", err)
	}

	if err := util.WaitCRDReady(p.CrdClient(), promCRDName, p.options.WaitInterval, p.options.WaitTimeout); err != nil {
		return fmt.Errorf("wait CRD %s ready fail, %w", promCRDName, err)
	}

	logrus.Infof("CRD is ready, start to install thanos")

	if _, err := karmadautil.EnsureNamespaceExist(p.KubeClient(), monitoringNamespace, false); err != nil {
		return fmt.Errorf("failed to ensure namespace %s, %w", monitoringNamespace, err)
	}
	s, err := objectStoreSecret(p.args.ObjectStoreConfig)
	if err != nil {
		return err
	}
	s.Namespace = monitoringNamespace

	if err := p.UpdateResource(s); err != nil {
		return fmt.Errorf("create object store config for prometheus fail, %w", err)
	}
	objectStorePolicy := &policyv1alpha1.PropagationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy.karmada.io/v1alpha1",
			Kind:       "PropagationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thanos-objstore-config",
			Namespace: monitoringNamespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: "v1",
					Kind:       "Secret",
					Name:       s.Name,
				},
			},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: p.clusters,
				},
			},
		},
	}
	if err := p.UpdateResource(objectStorePolicy); err != nil {
		return fmt.Errorf("create PropagationPolicy fail, %w", err)
	}

	logrus.Infof("Object store config is ready, start to install thanos sidecar")
	resourceList, err := p.resources(helmClient, promDir)
	if err != nil {
		return fmt.Errorf("load resource fail, %w", err)
	}
	if _, err := helmClient.Update(resourceList, resourceList, true); err != nil {
		return fmt.Errorf("create resource fail, %w", err)
	}

	// merge all resources before generate policy
	for _, r := range setupResourceList {
		resourceList.Append(r)
	}
	cpp, pp := p.generatePropagationPolicies(resourceList)

	if err := p.UpdateResource(cpp); err != nil {
		return fmt.Errorf("create ClusterPropagationPolicy fail, %w", err)
	}

	// wait apiEnablement in allCluster
	if err := util.WaitAPIEnableInClusters(p.KarmadaClient(), prometheusGVK, p.clusters, p.options.WaitInterval, p.options.WaitTimeout); err != nil {
		return fmt.Errorf("wait CRD %s ready fail, %w", promCRDName, err)
	}
	logrus.Debugf("thanos API enabled in all clusters")

	if err := p.UpdateResource(pp); err != nil {
		return fmt.Errorf("create PropagationPoliciy fail, %w", err)
	}

	if err := p.overrideExternalLabels(); err != nil {
		return err
	}

	if err := p.exposeThanosSidecar(); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) overrideExternalLabels() error {
	for _, c := range p.clusters {
		op := clusterOverridePolicy(c)
		if err := p.UpdateResource(op); err != nil {
			return fmt.Errorf("create OverridePolicy fail, %w", err)
		}
	}

	return nil
}

// runInstallThanos install thanos query and store in Host cluster
func (p *Plugin) runInstallThanos(namespace string) error {
	opts := generic.New()
	opts.KubeConfig = p.args.HostKubeconfig
	opts.KubeContext = p.args.HostContext

	host, err := client.NewClient(opts.RESTClientGetter())
	if err != nil {
		return err
	}

	if _, err := karmadautil.EnsureNamespaceExist(host.KubeClient(), namespace, false); err != nil {
		return fmt.Errorf("failed to ensure namespace %s, %w", namespace, err)
	}

	s, err := objectStoreSecret(p.args.ObjectStoreConfig)
	if err != nil {
		return err
	}
	s.Namespace = namespace

	if err := host.UpdateResource(s); err != nil {
		return fmt.Errorf("create object store config fail, %w", err)
	}

	helmClient := host.HelmClient()
	helmClient.Namespace = namespace
	resourceList, err := p.resources(helmClient, thanosDir)
	if err != nil {
		return err
	}

	if p.options.DryRun {
		logrus.Infof("apply resoucrs: %v", resourceList)
		return nil
	}

	logrus.Debugf("apply thanos manifests")
	// first delete existing resources to make install idempotent
	if _, err := helmClient.Update(resourceList, resourceList, true); err != nil {
		return err
	}

	// kubectl patch svc thanos-query -n thanos -p '{"spec": {"type": "LoadBalancer"}}'
	patch := `[{"op": "replace", "path": "/spec/type", "value":"LoadBalancer"}]`
	if _, err = host.KubeClient().CoreV1().Services(namespace).Patch(context.TODO(), thanosQuerySvcName, types.JSONPatchType,
		[]byte(patch), metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("patching service %s: %v", thanosQuerySvcName, err)
	}

	logrus.Debugf("wait for thanos resources ready")
	if err := helmClient.Wait(resourceList, p.options.WaitTimeout); err != nil {
		return fmt.Errorf("wait thanos resources timeout, %w", err)
	}

	logrus.Infof("Thanos resources applied, start to discovery thanos sidecar remote")
	if err := host.UpdateResource(thanosSidecarRemoteService); err != nil {
		return fmt.Errorf("create thanos sidecar remtoe service fail, %w", err)
	}

	ipList, err := p.thanosSidecarElbIPs()
	if err != nil {
		return fmt.Errorf("get thanos sidecar elb ip fail, %w", err)
	}

	thanosSidecarRemoteEndpoints := thanosSidecarRemoteEndpoints(ipList)
	thanosSidecarRemoteEndpoints.Namespace = namespace
	if err := host.UpdateResource(thanosSidecarRemoteEndpoints); err != nil {
		return fmt.Errorf("create thanos sidecar remtoe endpoints fail, %w", err)
	}

	return err
}

func (p *Plugin) exposeThanosSidecar() error {
	s, err := p.KubeClient().CoreV1().Services(monitoringNamespace).Get(context.TODO(), "prometheus-thanos-thanos-sidecar", metav1.GetOptions{})
	if err != nil {
		return err
	}

	// 2. create elb service for thanos
	thanosSidecarElbSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      thanosSidecarELBSvcName,
			Namespace: monitoringNamespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "grpc",
					Protocol: corev1.ProtocolTCP,
					Port:     10901,
				},
				{
					Name:     "http",
					Protocol: corev1.ProtocolTCP,
					Port:     10902,
				},
			},
			Selector:        s.Spec.Selector,
			SessionAffinity: corev1.ServiceAffinityNone,
			Type:            corev1.ServiceTypeLoadBalancer,
		},
	}
	if err := p.UpdateResource(thanosSidecarElbSvc); err != nil {
		return err
	}

	pp := &policyv1alpha1.PropagationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy.karmada.io/v1alpha1",
			Kind:       "PropagationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      thanosSidecarElbSvc.Name,
			Namespace: thanosSidecarElbSvc.Namespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: "v1",
					Kind:       "Service",
					Name:       thanosSidecarElbSvc.Name,
					Namespace:  thanosSidecarElbSvc.Namespace,
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

func (p *Plugin) generatePropagationPolicies(resourceList helmclient.ResourceList) (
	*policyv1alpha1.ClusterPropagationPolicy,
	*policyv1alpha1.PropagationPolicy) {
	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy.karmada.io/v1alpha1",
			Kind:       "ClusterPropagationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: thanosPolicyName,
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
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy.karmada.io/v1alpha1",
			Kind:       "PropagationPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      thanosPolicyName,
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

func (p *Plugin) resources(helm *helmclient.Client, root string) (helmclient.ResourceList, error) {
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

		resourceList, err := helm.Build(bytes.NewBuffer(b), false)
		if err != nil {
			// should not happen
			continue
		}

		infos = append(infos, resourceList...)
	}

	return infos, nil
}

func (p *Plugin) thanosSidecarElbIPs() ([]string, error) {
	s, err := p.KubeClient().CoreV1().Services(monitoringNamespace).Get(context.TODO(), thanosSidecarELBSvcName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	ipList := sets.NewString()
	for _, ingress := range s.Status.LoadBalancer.Ingress {
		ipList.Insert(ingress.IP)
	}

	return ipList.UnsortedList(), nil
}
