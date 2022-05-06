package istio

import (
	"context"
	"fmt"
	"time"

	karmadautil "github.com/karmada-io/karmada/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func waitIngressgatewayReady(gclient client.Client, cluster string,
	interval, timeout time.Duration) error {
	return waitPodReady(gclient, cluster, istioSystemNamespace, "app=istio-ingressgateway", interval, timeout)
}

func waitPodReady(gclient client.Client, cluster string, namespace, selector string, interval, timeout time.Duration) error {
	karmadaCluster, err := karmadautil.NewClusterClientSet(cluster, gclient, nil)
	if err != nil {
		return err
	}

	err = wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		pods, err := karmadaCluster.KubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
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
	if err != nil {
		return fmt.Errorf("ingressgateway in cluster %s not ready, err: %w", cluster, err)
	}

	return nil
}
