package volcano

import (
	"bytes"
	"context"

	policyv1alpha1 "github.com/karmada-io/karmada/pkg/apis/policy/v1alpha1"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/zirain/ubrain/pkg/client"
	"github.com/zirain/ubrain/pkg/generic"
	"github.com/zirain/ubrain/pkg/util"
)

const (
	volcanoSystemNamespace = "volcano-system"
)

type InstallArgs struct {
	Clusters []string
}

type Plugin struct {
	*client.Client

	args     *InstallArgs
	settings *generic.Options
	getter   *util.BinaryGetter
}

func NewPlugin(s *generic.Options, args *InstallArgs) (*Plugin, error) {
	plugin := &Plugin{
		settings: s,
		args:     args,
		getter:   util.NewBinaryGetter(s),
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
	valcanoYaml, err := p.getter.Valcano()
	if err != nil {
		return err
	}

	resourceList, err := p.HelmClient().Build(bytes.NewBufferString(valcanoYaml), false)
	if err != nil {
		return err
	}

	cpp, pp := p.generatePolicy(resourceList)

	if p.settings.DryRun {
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

	_ = util.AppendResourceSelector(p.KubeClient().Discovery(), cpp, pp, resourceList)

	return cpp, pp
}
