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
	"time"

	"github.com/ghodss/yaml"
	"github.com/hashicorp/go-multierror"
	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	karmadautil "github.com/karmada-io/karmada/pkg/util"
	"istio.io/istio/operator/pkg/manifest"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/tools/clientcmd/api/latest"

	"github.com/zirain/ubrain/pkg/util"
)

const (
	remotePilotAddressServiceName = "istiod-elb"
	istioSystemNamespace          = "istio-system"
	istioOperatorNamespace        = "istio-operator"
	karmadaClusterNamespace       = "karmada-cluster"
	primaryCluster                = "primary"
	cacertsSecretName             = "cacerts"

	crdKind = "CustomResourceDefinition"

	checkInterval = 10 * time.Second
	checkTimeout  = 2 * time.Minute
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
	p.Infof("Begin to ensure namespaces")
	if _, err := karmadautil.EnsureNamespaceExist(p.KubeClient(), istioSystemNamespace, false); err != nil {
		return fmt.Errorf("failed to ersure namespace %s, %w", istioSystemNamespace, err)
	}

	if _, err := karmadautil.EnsureNamespaceExist(p.KubeClient(), istioOperatorNamespace, false); err != nil {
		return fmt.Errorf("failed to ersure namespace %s, %w", istioOperatorNamespace, err)
	}

	return nil
}

func (p *IstioPlugin) createIstioCacerts() error {
	_, err := p.KubeClient().CoreV1().Secrets(istioSystemNamespace).Get(context.TODO(), cacertsSecretName, metav1.GetOptions{})
	if err == nil {
		// skip create cacerts if exists
		p.Infof("%s secret already exists, skipping create", cacertsSecretName)
		return nil
	}

	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("unexpect error when get %s secret, %w", cacertsSecretName, err)
	}

	// cacert not exists, begin to create
	p.Infof("Begin to create %s secret", cacertsSecretName)
	caCert, _ := os.ReadFile(path.Join(p.args.Cacerts, "ca-cert.pem"))
	caKey, _ := os.ReadFile(path.Join(p.args.Cacerts, "ca-key.pem"))
	rootCert, _ := os.ReadFile(path.Join(p.args.Cacerts, "root-cert.pem"))
	certChain, _ := os.ReadFile(path.Join(p.args.Cacerts, "cert-chain.pem"))

	cacert := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cacertsSecretName,
			Namespace: istioSystemNamespace,
		},
		StringData: map[string]string{
			"ca-cert.pem":    string(caCert),
			"ca-key.pem":     string(caKey),
			"root-cert.pem":  string(rootCert),
			"cert-chain.pem": string(certChain),
		},
	}

	if _, err := p.KubeClient().CoreV1().Secrets(istioSystemNamespace).
		Create(context.TODO(), cacert, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create secret %s, %w", cacertsSecretName, err)
	}

	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cacertsSecretName,
			Namespace: istioSystemNamespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: "v1",
					Kind:       "Secret",
					Namespace:  istioSystemNamespace,
					Name:       cacertsSecretName,
				},
			},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: p.allClusters(),
				},
			},
		},
	}

	_, err = p.KarmadaClient().PolicyV1alpha1().ClusterPropagationPolicies().Create(context.TODO(), cpp, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster propagation policy for secret %s, %w", cacertsSecretName, err)
	}
	return nil
}

func (p *IstioPlugin) installCrds() error {
	p.Infof("Begin to install crds in karmada-apiserver")
	// install crds in karmada-apiserver
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
		p.Infof("%s", string(out))
		return err
	}

	tmpYamlFile := path.Join(p.settings.TempDir, "manifest.yaml")
	if err = os.WriteFile(tmpYamlFile, out, 0644); err != nil {
		return err
	}

	crdFilter := func(r *resource.Info) bool {
		// only install crds here
		return r.Mapping.GroupVersionKind.Kind == crdKind
	}

	if err := p.applyWithFilter(out, crdFilter); err != nil {
		return err
	}

	if err := p.createIstioConfigClusterPropagationPolicy(); err != nil {
		return nil
	}

	return nil
}

func (p *IstioPlugin) createIstioConfigClusterPropagationPolicy() error {
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
			Name: "istio-config",
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

	_, err = p.KarmadaClient().PolicyV1alpha1().ClusterPropagationPolicies().Create(context.TODO(), cpp, metav1.CreateOptions{})

	return err
}

func (p *IstioPlugin) createIstioOperator() error {
	p.Infof("Begin to create istio operator deployment")
	if err := p.createIstioOperatorDeployment(); err != nil {
		return err
	}

	clusters := p.allClusters()
	// create ClusterPropagationPolicy for istio-operator's ClusterRole/ClusterRoleBinding/CustomResourceDefinition
	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "istio-operator",
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: "rbac.authorization.k8s.io/v1",
					Kind:       "ClusterRole",
					Name:       "istio-operator",
				},
				{
					APIVersion: "rbac.authorization.k8s.io/v1",
					Kind:       "ClusterRoleBinding",
					Name:       "istio-operator",
				},
				{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "CustomResourceDefinition",
					Name:       "istiooperators.install.istio.io",
				},
			},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: clusters,
				},
			},
		},
	}
	if _, err := p.KarmadaClient().PolicyV1alpha1().ClusterPropagationPolicies().Create(context.TODO(), cpp, metav1.CreateOptions{}); err != nil {
		return err
	}

	// create PropagationPolicy for istio-operator's Deployment/ServcieAccount
	cp := &policyv1alpha1.PropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-operator",
			Namespace: istioOperatorNamespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "istio-operator",
				},
				{
					APIVersion: "v1",
					Kind:       "ServiceAccount",
					Name:       "istio-operator",
				},
			},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: clusters,
				},
			},
		},
	}
	if _, err := p.KarmadaClient().PolicyV1alpha1().
		PropagationPolicies(cp.Namespace).Create(context.TODO(), cp, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (p *IstioPlugin) createIstioOperatorDeployment() error {
	cmd := exec.Command(p.istioctl, "operator", "dump")
	out, err := cmd.CombinedOutput()
	if err != nil {
		p.Infof("%s", string(out))
		return err
	}

	tmpYamlFile := path.Join(p.settings.TempDir, "istio-operator.yaml")
	if err = os.WriteFile(tmpYamlFile, out, 0644); err != nil {
		return err
	}

	excludeCrdFilter := func(r *resource.Info) bool {
		return r.Mapping.GroupVersionKind.Kind != crdKind
	}

	if err := p.applyWithFilter(out, excludeCrdFilter); err != nil {
		return err
	}

	return nil
}

func (p *IstioPlugin) installControlPlane() error {
	p.Infof("Begin to install istio control-plane on %s", p.args.Primary)
	if err := p.createIstioElb(); err != nil {
		return err
	}

	if err := p.createPrimaryIstioOperator(); err != nil {
		return err
	}
	if err := waitIngressgatewayReady(p.Client, p.args.Primary,
		checkInterval, checkTimeout); err != nil {
		return fmt.Errorf("istio control plane in cluster %s not ready, err: %w", p.args.Primary, err)
	}

	return nil
}

func (p *IstioPlugin) createIstioElb() error {
	istioElbSvc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istiod-elb",
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

	// create PropagationPolicy for istio-operator's Deployment/ServcieAccount
	cp := &policyv1alpha1.PropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      primaryCluster,
			Namespace: istioSystemNamespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: "v1",
					Kind:       "Service",
					Name:       "istiod-elb",
				},
				{
					APIVersion: "install.istio.io/v1alpha1",
					Kind:       "IstioOperator",
					Name:       primaryCluster,
				},
			},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: []string{p.args.Primary},
				},
			},
		},
	}
	if _, err := p.KarmadaClient().PolicyV1alpha1().
		PropagationPolicies(cp.Namespace).Create(context.TODO(), cp, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (p *IstioPlugin) createPrimaryIstioOperator() error {
	setFlags := make([]string, 0, len(p.args.SetFlags)+3)
	setFlags = append(setFlags, fmt.Sprintf("hub=%s", p.args.Hub))
	setFlags = append(setFlags, fmt.Sprintf("tag=%s", p.args.Tag))
	setFlags = append(setFlags, p.args.SetFlags...)
	// override clusterName to primary
	setFlags = append(setFlags, fmt.Sprintf("values.global.multiCluster.clusterName=%s", primaryCluster))

	data, iop, err := manifest.GenerateConfig(p.args.IopFiles, setFlags, false, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}
	iop.Name = primaryCluster
	iop.Namespace = istioSystemNamespace

	if err := p.apply([]byte(data)); err != nil {
		return fmt.Errorf("failed to create iop in primary cluster, %w", err)
	}

	return nil
}

func (p *IstioPlugin) installRemotes(remotePilotAddress string) error {
	var (
		wg       = sync.WaitGroup{}
		multiErr *multierror.Error
	)

	for _, remote := range p.args.Remotes {
		wg.Add(1)
		p.Infof("Begin to install istio in cluster %s", remote)

		if err := p.createIstioRemoteSecret(remote); err != nil {
			return nil
		}

		if err := p.createRemoteIstioOperator(remote, remotePilotAddress); err != nil {
			return err
		}

		go func(cluster string) {
			err := waitIngressgatewayReady(p.Client, cluster, checkInterval, checkTimeout)
			if err != nil {
				multierror.Append(multiErr, err)
			}
			wg.Done()
		}(remote)
	}
	wg.Wait()

	return multiErr.ErrorOrNil()
}

func (p *IstioPlugin) createIstioRemoteSecret(remote string) error {
	// create istio-remote-secret for remote cluster, it will be propagate to primary cluste
	istioRemoteSecret, err := p.generatrRemoteSecret(remote)
	if err != nil {
		return err
	}
	if _, err := p.KubeClient().CoreV1().Secrets(istioSystemNamespace).Create(context.TODO(), istioRemoteSecret, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create secret %s/%s, %w", istioOperatorNamespace, istioRemoteSecret.Name, err)
	}

	// create PropagationPolicy for istio-operator
	secretPolicy := &policyv1alpha1.PropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      istioRemoteSecret.Name,
			Namespace: istioSystemNamespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: "v1",
					Kind:       "Secret",
					Name:       istioRemoteSecret.Name,
				},
			},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: []string{p.args.Primary},
				},
			},
		},
	}
	if _, err := p.KarmadaClient().PolicyV1alpha1().PropagationPolicies(secretPolicy.Namespace).
		Create(context.TODO(), secretPolicy, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (p *IstioPlugin) createRemoteIstioOperator(remote string, remotePilotAddress string) error {
	setFlags := make([]string, 0, len(p.args.SetFlags)+2)
	setFlags = append(setFlags, fmt.Sprintf("hub=%s", p.args.Hub))
	setFlags = append(setFlags, fmt.Sprintf("tag=%s", p.args.Tag))
	setFlags = append(setFlags, p.args.SetFlags...)
	setFlags = append(setFlags, fmt.Sprintf("values.global.multiCluster.clusterName=%s", remote))
	setFlags = append(setFlags, fmt.Sprintf("values.global.remotePilotAddress=%s", remotePilotAddress))

	// use manifest merge IOP file with set flag, this should be safe for different version
	_, iop, err := manifest.GenerateConfig(p.args.IopFiles, setFlags, false, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}
	iop.Name = remote
	iop.Namespace = istioSystemNamespace

	// TODO: replace with istio client-go to avoid marshal/unmarshal
	b, err := yaml.Marshal(iop)
	if err != nil {
		return fmt.Errorf("failed to marshal istio operator, %w", err)
	}

	if err := p.apply(b); err != nil {
		return fmt.Errorf("failed to create iop in cluster %s, %w", remote, err)
	}

	// create PropagationPolicy for istio-operator
	cp := &policyv1alpha1.PropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("istio-%s", remote),
			Namespace: istioSystemNamespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: "install.istio.io/v1alpha1",
					Kind:       "IstioOperator",
					Name:       remote,
				},
			},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: []string{remote},
				},
			},
		},
	}
	if _, err := p.KarmadaClient().PolicyV1alpha1().
		PropagationPolicies(cp.Namespace).Create(context.TODO(), cp, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (p *IstioPlugin) remotePilotAddress() (string, error) {
	svc, err := p.KubeClient().CoreV1().Services(istioSystemNamespace).Get(context.TODO(), remotePilotAddressServiceName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	for _, ingress := range svc.Status.LoadBalancer.Ingress {
		if ingress.Hostname == p.args.Primary {
			return ingress.IP, nil
		}
	}

	return "", fmt.Errorf("service istiod-elb not found")
}

func (p *IstioPlugin) apply(manifest []byte) error {
	return p.applyWithFilter(manifest, nil)
}

func (p *IstioPlugin) applyWithFilter(manifest []byte, fn func(*resource.Info) bool) error {
	resource, err := p.HelmClient().Build(bytes.NewBuffer(manifest), false)
	if err != nil {
		return err
	}

	if fn != nil {
		resource = resource.Filter(fn)
	}

	if _, err := p.HelmClient().Create(resource); err != nil {
		return err
	}

	return nil
}

func (p *IstioPlugin) generatrRemoteSecret(remote string) (*v1.Secret, error) {
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
