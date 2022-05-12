package util

import (
	"context"
	"time"

	"github.com/zirain/ubrain/pkg/client"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd/api"
)

func CreateBearerTokenKubeconfig(caData, token []byte, clusterName, server string) *api.Config {
	c := &api.Config{
		Clusters: map[string]*api.Cluster{
			clusterName: {
				CertificateAuthorityData: caData,
				Server:                   server,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{},
		Contexts: map[string]*api.Context{
			clusterName: {
				Cluster:  clusterName,
				AuthInfo: clusterName,
			},
		},
		CurrentContext: clusterName,
	}

	c.AuthInfos[c.CurrentContext] = &api.AuthInfo{
		Token: string(token),
	}
	return c
}

func WaitPodReady(client *client.Client, cluster string, namespace, selector string, interval, timeout time.Duration) error {
	kubeClient, err := client.NewClusterClientSet(cluster)
	if err != nil {
		return err
	}

	err = wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		pods, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			return false, nil
		}

		if len(pods.Items) == 0 {
			return false, nil
		}

		for _, p := range pods.Items {
			if p.Status.Phase != v1.PodRunning {
				return false, nil
			}
		}

		return true, nil
	})

	return err
}
