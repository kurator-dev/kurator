package util

import (
	karmadaclientset "github.com/karmada-io/karmada/pkg/generated/clientset/versioned"
	"github.com/karmada-io/karmada/pkg/util/gclient"
	"helm.sh/helm/v3/pkg/kube"
	crdclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kubeclient "k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	Karmada karmadaclientset.Interface
	Kube    kubeclient.Interface
	Crd     crdclientset.Interface
	Helm    kube.Interface

	GlobalClient client.Client
}

func NewClient(rest genericclioptions.RESTClientGetter) (*Client, error) {
	kubeClient := kube.New(rest)

	cs, err := kubeClient.Factory.KubernetesClientSet()
	if err != nil {
		return nil, err
	}

	c, err := rest.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	crdClientSet, err := crdclientset.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	karmadaClient, err := karmadaclientset.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	gclient, err := gclient.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	return &Client{
		Kube:         cs,
		Helm:         kubeClient,
		Crd:          crdClientSet,
		Karmada:      karmadaClient,
		GlobalClient: gclient,
	}, nil
}
