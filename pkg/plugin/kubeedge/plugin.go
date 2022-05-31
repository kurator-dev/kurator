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

package kubeedge

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	karmadautil "github.com/karmada-io/karmada/pkg/util"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/moreos"
	"kurator.dev/kurator/pkg/util"
)

const (
	defaultPolicyName       = "kubeedge"
	cloudcoreELBServiceName = "cloudcore-elb"
)

var (
	keadmBinary      = filepath.Join("keadm" + moreos.Exe)
	metadataAccessor = meta.NewAccessor()
)

type InstallArgs struct {
	Clusters  []string
	Namespace string

	AdvertiseAddress string
}

type KubeEdgePlugin struct {
	*client.Client

	installArgs *InstallArgs

	options *generic.Options

	keadm string
}

func NewKubeEdgePlugin(opts *generic.Options, args *InstallArgs) (*KubeEdgePlugin, error) {
	plugin := &KubeEdgePlugin{
		installArgs: args,
		options:     opts,
		keadm:       "/usr/local/bin/keadm",
	}

	rest := opts.RESTClientGetter()
	c, err := client.NewClient(rest)
	if err != nil {
		return nil, err
	}
	plugin.Client = c

	return plugin, nil
}

// Execute receives an executable's filepath, a slice
// of arguments, and a slice of environment variables
// to relay to the executable.
func (p *KubeEdgePlugin) Execute(cmdArgs, environment []string) error {
	// download keadm
	keadmPath, err := p.installKeadm()
	if err == nil {
		p.keadm = keadmPath
	}

	if _, err := karmadautil.EnsureNamespaceExist(p.KubeClient(), p.installArgs.Namespace, p.options.DryRun); err != nil {
		return fmt.Errorf("failed to ensure namespace %s, %w", p.installArgs.Namespace, err)
	}

	if err := p.runInstall(); err != nil {
		logrus.Errorf("failed to install KubeEdge, %s", err)
		return err
	}

	if err := p.exposeCloudcore(); err != nil {
		return err
	}

	if err := p.checkReady(); err != nil {
		logrus.Errorf("check KubeEdge status failed, %s", err)
		return err
	}

	return nil
}

func (p *KubeEdgePlugin) runInstall() error {
	resources, err := p.generateKubeResources()
	if err != nil {
		return err
	}

	// create ClusterPropagationPolicy for kubeedge's cluster scoped resources
	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultPolicyName,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: p.installArgs.Clusters,
				},
			},
		},
	}

	// create PropagationPolicy for kubeedge's namespace scoped resources
	pp := &policyv1alpha1.PropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultPolicyName,
			Namespace: p.installArgs.Namespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: p.installArgs.Clusters,
				},
			},
		},
	}

	util.AppendResourceSelector(cpp, pp, resources)

	if p.options.DryRun {
		for _, r := range resources {
			logrus.Infof("%+v", r)
		}
		logrus.Infof("ClusterPropagationPolicy: %+v", cpp)
		logrus.Infof("PropagationPolicy: %+v", pp)
		return nil
	}

	if _, err := p.HelmClient().Create(resources); err != nil {
		return fmt.Errorf("run helm create failed, %w", err)
	}

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

func (p *KubeEdgePlugin) generateKubeResources() (kube.ResourceList, error) {
	installArgs := []string{
		"beta",
		"manifest",
		"generate",
	}
	if len(p.installArgs.AdvertiseAddress) != 0 {
		installArgs = append(installArgs, fmt.Sprintf("--advertise-address=%s", p.installArgs.AdvertiseAddress))
	}

	logrus.Debugf("run cmd: %s %v", p.keadm, installArgs)
	cmd := exec.Command(p.keadm, installArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Errorf("failed to generate manifest %s", string(out))
		return nil, err
	}

	resources, err := p.HelmClient().Build(bytes.NewBuffer(out), false)
	if err != nil {
		logrus.Errorf("failed to build helm resource %s", string(out))
		return nil, err
	}

	// resources generated by `keadm` are non-namespaced, set namespace before apply
	// more details https://github.com/kubeedge/kubeedge/issues/3843
	for _, r := range resources {
		if r.Namespaced() {
			r.Namespace = p.installArgs.Namespace
			metadataAccessor.SetNamespace(r.Object, p.installArgs.Namespace)
		}
	}

	return resources, nil
}

func (p *KubeEdgePlugin) installKeadm() (string, error) {
	kubeedgeComponent := p.options.Components["kubeedge"]
	ver := kubeedgeComponent.Version
	if !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}

	installPath := filepath.Join(p.options.HomeDir, kubeedgeComponent.Name, kubeedgeComponent.Version)
	keadmPath := filepath.Join(installPath, fmt.Sprintf("keadm-%s-%s-%s/keadm", ver, util.OSExt(), runtime.GOARCH), keadmBinary)
	_, err := os.Stat(keadmPath)
	if err == nil {
		return keadmPath, nil
	}

	if os.IsNotExist(err) {
		if err = os.MkdirAll(installPath, 0o750); err != nil {
			return "", fmt.Errorf("unable to create directory %q: %w", installPath, err)
		}

		// https://github.com/kubeedge/kubeedge/releases/download/v1.9.2/keadm-v1.9.2-linux-amd64.tar.gz
		url, _ := util.JoinUrlPath(kubeedgeComponent.ReleaseURLPrefix, ver,
			fmt.Sprintf("keadm-%s-%s-%s.tar.gz", ver, util.OSExt(), runtime.GOARCH))
		if _, err = util.DownloadResource(url, installPath); err != nil {
			return "", fmt.Errorf("unable to get keadm binary %q: %w", url, err)
		}
	}
	return util.VerifyExecutableBinary(keadmPath)
}

func (p *KubeEdgePlugin) checkReady() error {
	var (
		wg       = sync.WaitGroup{}
		multiErr *multierror.Error
	)

	for _, c := range p.installArgs.Clusters {
		wg.Add(1)
		go func(cluster string) {
			defer wg.Done()
			if err := waitCloudcoreReady(p.Client, p.options, cluster, p.installArgs.Namespace); err != nil {
				multierror.Append(multiErr, err)
			}
		}(c)
	}
	wg.Wait()

	return multiErr.ErrorOrNil()
}

func (p *KubeEdgePlugin) exposeCloudcore() error {
	cloudcoreService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: cloudcoreELBServiceName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"k8s-app":  "kubeedge",
				"kubeedge": "cloudcore",
			},
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Name:     "cloudhub",
					Port:     10000,
					Protocol: "TCP",
				},
				{
					Name:     "cloudhub-https",
					Port:     10002,
					Protocol: "TCP",
				},
			},
		},
	}

	if _, err := p.Client.KubeClient().CoreV1().Services(p.installArgs.Namespace).Create(context.TODO(), cloudcoreService, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("expose cloudcore fail, %w", err)
	}

	pp := &policyv1alpha1.PropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cloudcoreELBServiceName,
			Namespace: p.installArgs.Namespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{
				{
					APIVersion: "v1",
					Kind:       "Service",
					Name:       cloudcoreELBServiceName,
					Namespace:  p.installArgs.Namespace,
				},
			},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: p.installArgs.Clusters,
				},
			},
		},
	}

	if _, err := p.KarmadaClient().PolicyV1alpha1().PropagationPolicies(pp.Namespace).
		Create(context.TODO(), pp, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}
