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

package istio

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	karmadautil "github.com/karmada-io/karmada/pkg/util"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/kube"
	"istio.io/istio/operator/pkg/manifest"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"sigs.k8s.io/yaml"

	"kurator.dev/kurator/pkg/cert"
	"kurator.dev/kurator/pkg/util"
)

const (
	remotePilotAddressServiceName = "istiod-elb"
	eastwestgatewayServiceName    = "istio-eastwestgateway"
	istioSystemNamespace          = "istio-system"
	istioOperatorNamespace        = "istio-operator"
	karmadaClusterNamespace       = "karmada-cluster"

	iopCRDName = "istiooperators.install.istio.io"
	crdKind    = "CustomResourceDefinition"
)

func (p *IstioPlugin) runInstall() error {
	if err := p.ensureNamespaces(); err != nil {
		return err
	}

	if err := p.installCrds(); err != nil {
		return err
	}

	if err := p.createIstioCacerts(); err != nil {
		return err
	}

	if err := p.createIstioOperator(); err != nil {
		return err
	}

	if err := p.installControlPlane(); err != nil {
		return err
	}

	pilotAddress, err := p.remotePilotAddress()
	if err != nil {
		return err
	}

	if err := p.installRemotes(pilotAddress); err != nil {
		return err
	}

	return nil
}

func (p *IstioPlugin) ensureNamespaces() error {
	logrus.Infof("Begin to ensure namespaces")
	if _, err := karmadautil.EnsureNamespaceExist(p.KubeClient(), istioSystemNamespace, false); err != nil {
		return fmt.Errorf("failed to ersure namespace %s, %w", istioSystemNamespace, err)
	}

	if _, err := karmadautil.EnsureNamespaceExist(p.KubeClient(), istioOperatorNamespace, false); err != nil {
		return fmt.Errorf("failed to ersure namespace %s, %w", istioOperatorNamespace, err)
	}

	return nil
}

func (p *IstioPlugin) createIstioCacerts() error {
	logrus.Infof("Begin to create istio cacerts")
	var gen cert.Generator
	if len(p.args.Cacerts) != 0 {
		gen = cert.NewPluggedCert(p.args.Cacerts)
	} else {
		gen = cert.NewSelfSignedCert("cluster.local")
	}
	cacert, err := gen.Secret(istioSystemNamespace)
	if err != nil {
		return fmt.Errorf("failed to gen secret, %w", err)
	}

	_, err = p.KubeClient().CoreV1().Secrets(cacert.Namespace).Get(context.TODO(), cacert.Name, metav1.GetOptions{})
	if err == nil {
		// skip create cacerts if exists
		logrus.Infof("secret %s/%s already exists, skipping create", cacert.Namespace, cacert.Name)
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("unexpect error when get secret %s/%s, %w", cacert.Namespace, cacert.Name, err)
	}

	if _, err := p.KubeClient().CoreV1().Secrets(cacert.Namespace).
		Create(context.TODO(), cacert, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create secret %s/%s, %w", cacert.Namespace, cacert.Name, err)
	}

	return util.CreatePropagationPolicy(p.KarmadaClient(), p.allClusters(), cacert)
}

func (p *IstioPlugin) installCrds() error {
	logrus.Infof("Begin to install istio crds in karmada-apiserver and primary cluster")
	args := []string{
		"profile=external",
		"values.global.configCluster=true",
		"values.global.externalIstiod=false",
		"values.global.defaultPodDisruptionBudget.enabled=false",
		"values.telemetry.enabled=false",
	}
	istioctlArgs := make([]string, 0, 2*len(args)+2)
	istioctlArgs = append(istioctlArgs, "manifest", "generate")
	for _, arg := range args {
		istioctlArgs = append(istioctlArgs, "--set")
		istioctlArgs = append(istioctlArgs, arg)
	}

	cmd := exec.Command(p.istioctl, istioctlArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Infof("%s", string(out))
		return err
	}

	tmpYamlFile := path.Join(p.options.HomeDir, p.options.Components["istio"].Name, "manifest.yaml")
	if err = os.WriteFile(tmpYamlFile, out, 0644); err != nil {
		return err
	}

	crdFilter := func(r *resource.Info) bool {
		// only install crds here
		// istiooperators will be install in createIstioOperator, exclude it to avoid AlreadyExists error.
		return r.Mapping.GroupVersionKind.Kind == crdKind && r.Name != iopCRDName
	}

	if _, err := p.applyWithFilter(out, crdFilter); err != nil {
		return err
	}

	if err := p.createIstioCustomResourceClusterPropagationPolicy(); err != nil {
		return nil
	}

	return nil
}

func (p *IstioPlugin) createIstioCustomResourceClusterPropagationPolicy() error {
	crds, err := p.CrdClient().ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list crds, %w", err)
	}

	resourceSelectors := make([]policyv1alpha1.ResourceSelector, 0)
	for _, crd := range crds.Items {
		if !strings.HasSuffix(crd.Name, "istio.io") {
			continue
		}

		g := crd.Spec.Group
		for _, ver := range crd.Spec.Versions {
			s := policyv1alpha1.ResourceSelector{
				APIVersion: fmt.Sprintf("%s/%s", g, ver.Name),
				Kind:       crd.Spec.Names.Kind,
			}

			resourceSelectors = append(resourceSelectors, s)
		}
	}

	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "istio-customresource-to-primary",
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: resourceSelectors,
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: []string{p.args.Primary},
				},
			},
		},
	}

	_, err = p.KarmadaClient().PolicyV1alpha1().ClusterPropagationPolicies().
		Create(context.TODO(), cpp, metav1.CreateOptions{})

	return err
}

func (p *IstioPlugin) createIstioOperator() error {
	logrus.Infof("Begin to create istio operator deployment")
	resources, err := p.createIstioOperatorDeployment()
	if err != nil {
		return err
	}

	clusters := p.allClusters()

	// create ClusterPropagationPolicy for istio-operator's ClusterRole/ClusterRoleBinding
	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "istio-operator",
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: clusters,
				},
			},
		},
	}

	// create PropagationPolicy for istio-operator's Deployment/ServiceAccount
	pp := &policyv1alpha1.PropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-operator",
			Namespace: istioOperatorNamespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: clusters,
				},
			},
		},
	}

	util.AppendResourceSelector(cpp, pp, resources)

	if _, err := p.KarmadaClient().PolicyV1alpha1().ClusterPropagationPolicies().
		Create(context.TODO(), cpp, metav1.CreateOptions{}); err != nil {
		return err
	}

	if _, err := p.KarmadaClient().PolicyV1alpha1().PropagationPolicies(pp.Namespace).
		Create(context.TODO(), pp, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (p *IstioPlugin) createIstioOperatorDeployment() (kube.ResourceList, error) {
	cmd := exec.Command(p.istioctl, "operator", "dump")
	out, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Infof("%s", string(out))
		return nil, err
	}

	tmpYamlFile := path.Join(p.options.HomeDir, p.options.Components["istio"].Name, "istio-operator.yaml")
	if err = os.WriteFile(tmpYamlFile, out, 0644); err != nil {
		return nil, err
	}

	resources, err := p.apply(out)
	if err != nil {
		return resources, err
	}

	return resources, nil
}

func (p *IstioPlugin) installControlPlane() error {
	logrus.Infof("Begin to install istio control-plane on %s", p.args.Primary)
	if err := p.createIstioElb(); err != nil {
		return err
	}

	if err := p.createPrimaryIstioOperator(); err != nil {
		return err
	}
	if err := waitIngressgatewayReady(p.Client, p.options, p.args.Primary); err != nil {
		return fmt.Errorf("istio control plane in cluster %s not ready, err: %w", p.args.Primary, err)
	}

	if err := p.exposeServices(); err != nil {
		return fmt.Errorf("expose service on %s fail, err: %w", p.args.Primary, err)
	}

	return nil
}

func (p *IstioPlugin) exposeServices() error {
	if p.IsFlat() {
		logrus.Debugf("skipping expose services in flat network")
		return nil
	}

	if err := waitEastwestgatewayReady(p.Client, p.options, p.args.Primary); err != nil {
		return fmt.Errorf("istio-eastwestgateway in cluster %s not ready, err: %w", p.args.Primary, err)
	}

	m, err := exposeServicesFiles()
	if err != nil {
		return err
	}

	_, err = p.apply([]byte(m))
	if err != nil {
		return err
	}

	return nil
}

func (p *IstioPlugin) createIstioElb() error {
	// in non-flat network mesh, we use eastwestgateway to export istiod
	if !p.IsFlat() {
		return nil
	}

	istioElbSvc := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      remotePilotAddressServiceName,
			Namespace: istioSystemNamespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     "tcp",
					Protocol: v1.ProtocolTCP,
					Port:     15012,
				},
			},
			Selector: map[string]string{
				"app":   "istiod",
				"istio": "pilot",
			},
			SessionAffinity: v1.ServiceAffinityNone,
			Type:            v1.ServiceTypeLoadBalancer,
		},
	}
	if _, err := p.KubeClient().CoreV1().
		Services(istioSystemNamespace).Create(context.TODO(), istioElbSvc, metav1.CreateOptions{}); err != nil {
		return err
	}

	return util.CreatePropagationPolicy(p.KarmadaClient(), []string{p.args.Primary}, istioElbSvc)
}

func (p *IstioPlugin) createPrimaryIstioOperator() error {
	setFlags := make([]string, 0, len(p.args.SetFlags)+3)
	// override hub and tag before user's flags
	setFlags = append(setFlags, fmt.Sprintf("hub=%s", p.args.Hub))
	setFlags = append(setFlags, fmt.Sprintf("tag=%s", p.args.Tag))
	setFlags = append(setFlags, p.args.SetFlags...)
	// override clusterName to primary, control plane cluster always named `primary`
	setFlags = append(setFlags, fmt.Sprintf("values.global.multiCluster.clusterName=%s", p.args.Primary))

	// TODO: always use eastwest-gateway on Primary?
	iopFiles, err := p.iopFiles(p.args.Primary)
	if err != nil {
		return fmt.Errorf("failed to get iop files: %w", err)
	}
	_, iop, err := manifest.GenerateConfig(iopFiles, setFlags, false, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}
	iop.Name = p.args.Primary
	iop.Namespace = istioSystemNamespace

	// TODO: replace this to avoid marshal/unmarshal once IstioOperator add to istio client-go
	b, err := yaml.Marshal(iop)
	if err != nil {
		return fmt.Errorf("failed to marshal istio operator, %w", err)
	}

	if _, err := p.apply(b); err != nil {
		return fmt.Errorf("failed to create iop in primary cluster, %w", err)
	}

	return util.CreatePropagationPolicy(p.KarmadaClient(), []string{p.args.Primary}, iop)
}

func (p *IstioPlugin) installRemotes(remotePilotAddress string) error {
	var (
		wg       = sync.WaitGroup{}
		multiErr *multierror.Error
	)

	for _, remote := range p.args.Remotes {
		logrus.Infof("Begin to install istio in cluster %s", remote)

		if err := p.createIstioRemoteSecret(remote); err != nil {
			return nil
		}

		if err := p.createRemoteIstioOperator(remote, remotePilotAddress); err != nil {
			return err
		}

		wg.Add(1)
		go func(cluster string) {
			defer wg.Done()
			err := waitIngressgatewayReady(p.Client, p.options, cluster)
			if err != nil {
				_ = multierror.Append(multiErr, err)
			}
		}(remote)
	}
	wg.Wait()

	return multiErr.ErrorOrNil()
}

func (p *IstioPlugin) networks() map[string][]string {
	m := map[string][]string{}
	for _, r := range p.allClusters() {
		k := networkName(r)
		m[k] = []string{r}
	}

	return m
}

func networkName(cluster string) string {
	return fmt.Sprintf("%s-network", cluster)
}

func (p *IstioPlugin) iopFiles(cluster string) ([]string, error) {
	iopFiles := make([]string, 0, len(p.args.IopFiles)+1)
	iopFiles = append(iopFiles, p.args.IopFiles...)
	if !p.IsFlat() {
		mesh := meshOptions{
			Network:     networkName(cluster), // TODO: support custom network, e.g. primary and remote1 in network1, remote2 in network2
			MeshID:      "mesh1",              // TODO: make this configurable
			ClusterName: cluster,
			Networks:    p.networks(),
		}

		logrus.Debugf("mesh options: %+v", mesh)
		b, err := templateEastWest(mesh)
		if err != nil {
			return nil, err
		}
		additionalIopFile := path.Join(p.options.HomeDir, fmt.Sprintf("eastwest-%s.yaml", cluster))
		if err := os.WriteFile(additionalIopFile, b, 0o644); err != nil {
			return nil, err
		}

		iopFiles = append(iopFiles, additionalIopFile)
	}

	return iopFiles, nil
}

func (p *IstioPlugin) createIstioRemoteSecret(remote string) error {
	// create istio-remote-secret for remote cluster, it will be propagated to remote cluster
	istioRemoteSecret, err := p.generateRemoteSecret(remote)
	if err != nil {
		return err
	}
	if _, err := p.KubeClient().CoreV1().Secrets(istioRemoteSecret.Namespace).Create(context.TODO(), istioRemoteSecret, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create secret %s/%s, %w", istioRemoteSecret.Namespace, istioRemoteSecret.Name, err)
	}

	// create PropagationPolicy for Secret
	return util.CreatePropagationPolicy(p.KarmadaClient(), []string{p.args.Primary}, istioRemoteSecret)
}

func (p *IstioPlugin) createRemoteIstioOperator(remote string, remotePilotAddress string) error {
	setFlags := make([]string, 0, len(p.args.SetFlags)+4)
	// override hub and tag before user's flags
	setFlags = append(setFlags, fmt.Sprintf("hub=%s", p.args.Hub))
	setFlags = append(setFlags, fmt.Sprintf("tag=%s", p.args.Tag))
	setFlags = append(setFlags, p.args.SetFlags...)
	setFlags = append(setFlags, fmt.Sprintf("values.global.multiCluster.clusterName=%s", remote))
	setFlags = append(setFlags, fmt.Sprintf("values.global.remotePilotAddress=%s", remotePilotAddress))

	iopFiles, err := p.iopFiles(remote)
	if err != nil {
		return fmt.Errorf("failed to get iop files: %w", err)
	}

	// use manifest merge IOP file with set flag, this should be safe for different version
	_, iop, err := manifest.GenerateConfig(iopFiles, setFlags, false, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}
	iop.Name = remote
	iop.Namespace = istioSystemNamespace

	// TODO: replace this to avoid marshal/unmarshal once IstioOperator add to istio client-go
	b, err := yaml.Marshal(iop)
	if err != nil {
		return fmt.Errorf("failed to marshal istio operator, %w", err)
	}

	if _, err := p.apply(b); err != nil {
		return fmt.Errorf("failed to create iop in cluster %s, %w", remote, err)
	}

	return util.CreatePropagationPolicy(p.KarmadaClient(), []string{remote}, iop)
}

func (p *IstioPlugin) remotePilotAddress() (string, error) {
	if !p.IsFlat() {
		return p.eastwestgatewayAddress()
	}

	svc, err := p.KubeClient().CoreV1().Services(istioSystemNamespace).Get(context.TODO(), remotePilotAddressServiceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", fmt.Errorf("service %s not found", remotePilotAddressServiceName)
		}
		return "", err
	}

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.Hostname == p.args.Primary {
			return ingress.IP, nil
		}
	}

	return "", fmt.Errorf("service %s status is pending", remotePilotAddressServiceName)
}

func (p *IstioPlugin) eastwestgatewayAddress() (string, error) {
	c, err := p.NewClusterClientSet(p.args.Primary)
	if err != nil {
		return "", err
	}
	svc, err := c.CoreV1().Services(istioSystemNamespace).Get(context.TODO(), eastwestgatewayServiceName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", fmt.Errorf("service %s not found", eastwestgatewayServiceName)
		}
		return "", err
	}

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			return ingress.IP, nil
		}
	}

	return "", fmt.Errorf("failed to get service %s status", eastwestgatewayServiceName)
}

func (p *IstioPlugin) apply(manifest []byte) (kube.ResourceList, error) {
	return p.applyWithFilter(manifest, nil)
}

func (p *IstioPlugin) applyWithFilter(manifest []byte, fn func(*resource.Info) bool) (kube.ResourceList, error) {
	resource, err := p.HelmClient().Build(bytes.NewBuffer(manifest), false)
	if err != nil {
		return nil, err
	}

	if fn != nil {
		resource = resource.Filter(fn)
	}

	if _, err := p.HelmClient().Create(resource); err != nil {
		return resource, err
	}

	return resource, nil
}

func (p *IstioPlugin) generateRemoteSecret(remote string) (*v1.Secret, error) {
	secret, err := p.KubeClient().CoreV1().Secrets(karmadaClusterNamespace).Get(context.TODO(), remote, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s, err: %w", remote, err)
	}

	cluster, err := p.KarmadaClient().ClusterV1alpha1().Clusters().Get(context.TODO(), remote, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster %s, err: %w", remote, err)
	}

	caBundle, ok := secret.Data["caBundle"]
	if !ok {
		return nil, fmt.Errorf("failed to get caBundle from secret %s", secret.Name)
	}

	token, ok := secret.Data["token"]
	if !ok {
		return nil, fmt.Errorf("failed to get token from secret %s", secret.Name)
	}

	kubeconfig := util.CreateBearerTokenKubeconfig(caBundle, token, remote, cluster.Spec.APIEndpoint)
	var data bytes.Buffer
	if err := latest.Codec.Encode(kubeconfig, &data); err != nil {
		return nil, err
	}

	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("istio-remote-secret-%s", remote),
			Namespace: istioSystemNamespace,
			Labels: map[string]string{
				"istio/multiCluster": "true",
			},
			Annotations: map[string]string{
				"networking.istio.io/cluster": remote,
			},
		},
		StringData: map[string]string{
			remote: data.String(),
		},
	}, nil
}

func (p *IstioPlugin) allClusters() []string {
	allClusters := make([]string, 0, len(p.args.Remotes)+1)
	allClusters = append(allClusters, p.args.Primary)
	allClusters = append(allClusters, p.args.Remotes...)
	return allClusters
}
