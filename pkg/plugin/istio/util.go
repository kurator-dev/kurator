package istio

import (
	"context"
	"time"

	"github.com/zirain/ubrain/pkg/client"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func waitIngressgatewayReady(client *client.Client, cluster string,
	interval, timeout time.Duration) error {
	return waitPodReady(client, cluster, istioSystemNamespace, "app=istio-ingressgateway", interval, timeout)
}

func waitPodReady(client *client.Client, cluster string, namespace, selector string, interval, timeout time.Duration) error {
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
