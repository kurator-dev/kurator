/*
Copyright 2022-2025 Kurator Authors.

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

package client

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"

	flaggerapi "github.com/fluxcd/flagger/pkg/apis/flagger/v1beta1"
	clusterv1alpha1 "github.com/karmada-io/karmada/pkg/apis/cluster/v1alpha1"
	karmadaclientset "github.com/karmada-io/karmada/pkg/generated/clientset/versioned"
	promclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/sirupsen/logrus"
	tektonapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	veleroapi "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	helmclient "helm.sh/helm/v3/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ingressv1 "k8s.io/api/networking/v1"
	crdclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type Client struct {
	kube kubeclient.Interface
	crd  crdclientset.Interface
	helm *helmclient.Client

	karmada karmadaclientset.Interface
	prom    promclient.Interface
	// it currently only support k8s core API, tekton API and velero API, because only these schemes are registered
	ctrlRuntimeClient client.Client
}

func NewClient(rest genericclioptions.RESTClientGetter) (*Client, error) {
	c, err := rest.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	kubeClient := kubeclient.NewForConfigOrDie(c)
	helmClient := helmclient.New(rest)
	crdClientSet := crdclientset.NewForConfigOrDie(c)
	karmadaClient := karmadaclientset.NewForConfigOrDie(c)
	promClient := promclient.NewForConfigOrDie(c)

	// create Scheme to add velero resource
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add corev1 to scheme: %v", err)
	}
	if err := veleroapi.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add veleroapi to scheme: %v", err)
	}
	if err := tektonapi.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add tektonapi to scheme: %v", err)
	}
	// add flagger resource
	if err := flaggerapi.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add flagger api to scheme: %v", err)
	}
	// add appsv1 resource
	// TODO: add commonly used resources for k8s
	if err := appsv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add appv1 api to scheme: %v", err)
	}
	if err := ingressv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add ingress api to scheme: %v", err)
	}
	// create controller-runtime client with scheme
	ctrlRuntimeClient, err := client.New(c, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return &Client{
		kube:              kubeClient,
		helm:              helmClient,
		crd:               crdClientSet,
		karmada:           karmadaClient,
		prom:              promClient,
		ctrlRuntimeClient: ctrlRuntimeClient,
	}, nil
}

func (c *Client) KubeClient() kubeclient.Interface {
	return c.kube
}

func (c *Client) KarmadaClient() karmadaclientset.Interface {
	return c.karmada
}

func (c *Client) CrdClient() crdclientset.Interface {
	return c.crd
}

func (c *Client) HelmClient() *helmclient.Client {
	return c.helm
}

func (c *Client) PromClient() promclient.Interface {
	return c.prom
}

func (c *Client) UpdateResource(obj interface{}) error {
	b, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	r, err := c.helm.Build(bytes.NewBuffer(b), false)
	if err != nil {
		return err
	}

	_, err = c.helm.Update(r, r, true)

	return err
}

// Note: partly copied from https://github.com/karmada-io/karmada/blob/592fa3224d48e5b5f60f9202bd5547df53ef4588/pkg/util/membercluster_client.go
// Refer to: karmada-io/karmada
func (c *Client) memberClusterConfig(clusterName string) (*rest.Config, error) {
	cluster, err := c.karmada.ClusterV1alpha1().Clusters().Get(context.TODO(), clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	apiEndpoint := cluster.Spec.APIEndpoint
	if apiEndpoint == "" {
		return nil, fmt.Errorf("the api endpoint of cluster %s is empty", clusterName)
	}

	secretNamespace := cluster.Spec.SecretRef.Namespace
	secretName := cluster.Spec.SecretRef.Name
	if secretName == "" {
		return nil, fmt.Errorf("cluster %s does not have a secret name", clusterName)
	}
	secret, err := c.kube.CoreV1().Secrets(secretNamespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	token, tokenFound := secret.Data[clusterv1alpha1.SecretTokenKey]
	if !tokenFound || len(token) == 0 {
		return nil, fmt.Errorf("the secret for cluster %s is missing a non-empty value for %q", clusterName, clusterv1alpha1.SecretTokenKey)
	}

	clusterConfig, err := clientcmd.BuildConfigFromFlags(apiEndpoint, "")
	if err != nil {
		return nil, err
	}

	clusterConfig.BearerToken = string(token)

	if cluster.Spec.InsecureSkipTLSVerification {
		clusterConfig.TLSClientConfig.Insecure = true
	} else {
		clusterConfig.CAData = secret.Data[clusterv1alpha1.SecretCADataKey]
	}

	if cluster.Spec.ProxyURL != "" {
		proxy, err := url.Parse(cluster.Spec.ProxyURL)
		if err != nil {
			logrus.Errorf("parse proxy error. %v", err)
			return nil, err
		}
		clusterConfig.Proxy = http.ProxyURL(proxy)
	}

	return clusterConfig, nil
}

func (c *Client) NewClusterClientSet(clusterName string) (kubeclient.Interface, error) {
	clusterConfig, err := c.memberClusterConfig(clusterName)
	if err != nil {
		return nil, err
	}
	return kubeclient.NewForConfig(clusterConfig)
}

func (c *Client) NewClusterCRDClientset(clusterName string) (crdclientset.Interface, error) {
	clusterConfig, err := c.memberClusterConfig(clusterName)
	if err != nil {
		return nil, err
	}
	return crdclientset.NewForConfig(clusterConfig)
}

func (c *Client) NewClusterHelmClient(clusterName string) (helmclient.Interface, error) {
	clusterConfig, err := c.memberClusterConfig(clusterName)
	if err != nil {
		return nil, err
	}

	clusterGetter := NewRESTClientGetter(clusterConfig)
	return helmclient.New(clusterGetter), nil
}

func (c *Client) CtrlRuntimeClient() client.Client {
	return c.ctrlRuntimeClient
}
