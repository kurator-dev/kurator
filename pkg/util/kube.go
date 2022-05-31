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

package util

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api"

	"kurator.dev/kurator/pkg/client"
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

// WaitMemberClusterPodReady return until the member cluster's pod is ready or an error occurs.
// Note: client is the karmada apiserver client, cluster should be a valid member cluster name.
func WaitMemberClusterPodReady(client *client.Client, cluster, namespace, selector string, interval, timeout time.Duration) error {
	kubeClient, err := client.NewClusterClientSet(cluster)
	if err != nil {
		return err
	}

	return WaitPodReady(kubeClient, namespace, selector, interval, timeout)
}

func WaitPodReady(client kubeclient.Interface, namespace, selector string, interval, timeout time.Duration) error {
	return wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		pods, err := client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			return false, nil
		}

		if len(pods.Items) == 0 {
			return false, nil
		}

		readyCount := 0
		for _, p := range pods.Items {
			for _, cond := range p.Status.Conditions {
				if cond.Type == v1.PodReady && cond.Status == v1.ConditionTrue {
					readyCount++
				}
			}
		}

		return readyCount == len(pods.Items), nil
	})
}
