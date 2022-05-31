package volcano

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"

	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"kurator.dev/kurator/pkg/client"
	"kurator.dev/kurator/pkg/generic"
	"kurator.dev/kurator/pkg/util"
)

const (
	volcanoSystemNamespace = "volcano-system"
)

type InstallArgs struct {
	Clusters []string
}

type Plugin struct {
	*client.Client

	args    *InstallArgs
	options *generic.Options
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
	valcanoYaml, err := p.volcanoManifest()
	if err != nil {
		return err
	}

	resourceList, err := p.HelmClient().Build(bytes.NewBufferString(valcanoYaml), false)
	if err != nil {
		return err
	}

	cpp, pp := p.generatePolicy(resourceList)

	if p.options.DryRun {
		logrus.Infof("apply resoucrs: %s", valcanoYaml)
		out, _ := yaml.Marshal(cpp)
		logrus.Infof("apply ClusterPropagationPolicy: %s", out)
		out, _ = yaml.Marshal(pp)
		logrus.Infof("apply PropagationPolicy: %s", out)
		return nil
	}

	if _, err := p.HelmClient().Create(resourceList); err != nil {
		return err
	}

	if _, err := p.KarmadaClient().PolicyV1alpha1().ClusterPropagationPolicies().Create(context.TODO(), cpp, metav1.CreateOptions{}); err != nil {
		return err
	}

	if _, err := p.KarmadaClient().PolicyV1alpha1().PropagationPolicies(pp.Namespace).Create(context.TODO(), pp, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

func (p *Plugin) generatePolicy(resourceList kube.ResourceList) (
	*policyv1alpha1.ClusterPropagationPolicy,
	*policyv1alpha1.PropagationPolicy) {
	cpp := &policyv1alpha1.ClusterPropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "volcano",
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: p.args.Clusters,
				},
			},
		},
	}

	pp := &policyv1alpha1.PropagationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "volcano",
			Namespace: volcanoSystemNamespace,
		},
		Spec: policyv1alpha1.PropagationSpec{
			ResourceSelectors: []policyv1alpha1.ResourceSelector{},
			Placement: policyv1alpha1.Placement{
				ClusterAffinity: &policyv1alpha1.ClusterAffinity{
					ClusterNames: p.args.Clusters,
				},
			},
		},
	}

	util.AppendResourceSelector(cpp, pp, resourceList)

	return cpp, pp
}

func (p *Plugin) volcanoManifest() (string, error) {
	volcano := p.options.Components["volcano"]

	// x84_64 https://raw.githubusercontent.com/volcano-sh/volcano/master/installer/volcano-development.yaml
	// arm64 https://raw.githubusercontent.com/volcano-sh/volcano/v1.5.1/installer/volcano-development.yaml
	ver := volcano.Version
	if ver != "master" && !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}

	var manifestName string
	// TODO: change it, the machine used to install volcano can be different from the destination cluster arch
	switch runtime.GOARCH {
	case "amd64":
		manifestName = "volcano-development.yaml"
	case "arm64":
		manifestName = "volcano-development-arm64.yaml"
	default:
		return "", fmt.Errorf("os arch %s is not supportted", runtime.GOARCH)
	}
	url, _ := util.JoinUrlPath(volcano.ReleaseURLPrefix, ver, manifestName)
	yaml, err := util.DownloadResource(url, "")
	if err != nil {
		return "", err
	}

	return yaml, nil
}
