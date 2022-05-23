package util

import (
	"context"
	"fmt"

	"github.com/karmada-io/karmada/pkg/apis/cluster/v1alpha1"
	karmadaclientset "github.com/karmada-io/karmada/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsClustersReady(karmada karmadaclientset.Interface, clusterNames []string) error {
	allClusters, err := karmada.ClusterV1alpha1().Clusters().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list karmada cluster fail, %w", err)
	}

	clusters := map[string]*v1alpha1.Cluster{}
	for _, c := range allClusters.Items {
		cluster := c
		clusters[c.Name] = &cluster
	}

	for _, c := range clusterNames {
		cluster, ok := clusters[c]
		if !ok {
			return fmt.Errorf("%s is not a valid cluster in karmada", c)
		}

		if !isReady(cluster) {
			return fmt.Errorf("status of %s is not valid", c)
		}
	}

	return nil
}

func isReady(cluster *v1alpha1.Cluster) bool {
	for _, cond := range cluster.Status.Conditions {
		if cond.Type == v1alpha1.ClusterConditionReady &&
			cond.Status == metav1.ConditionTrue {
			return true
		}
	}

	return false
}
