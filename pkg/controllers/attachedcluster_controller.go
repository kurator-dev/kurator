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

package controllers

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	clusterv1alpha1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
)

// AttachedClusterController reconciles a AttachedCluster object
type AttachedClusterController struct {
	client.Client
	APIReader client.Reader
	Scheme    *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (a *AttachedClusterController) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1alpha1.AttachedCluster{}).
		WithOptions(options).
		Complete(a)
}

func (a *AttachedClusterController) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)
	// Fetch the attachedCluster instance.
	attachedCluster := &clusterv1alpha1.AttachedCluster{}
	if err := a.Client.Get(ctx, req.NamespacedName, attachedCluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("attachedCluster is not exist", "attachedCluster", req)
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	log = log.WithValues("attachedCluster", klog.KObj(attachedCluster))
	ctx = ctrl.LoggerInto(ctx, log)
	patchHelper, err := patch.NewHelper(attachedCluster, a.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to init patch helper for fleet %s", req.NamespacedName)
	}

	defer func() {
		patchOpts := []patch.Option{}
		if err := patchHelper.Patch(ctx, attachedCluster, patchOpts...); err != nil {
			reterr = utilerrors.NewAggregate([]error{reterr, errors.Wrapf(err, "failed to patch fleet %s", req.NamespacedName)})
		}
	}()

	return a.reconcile(ctx, attachedCluster)
}

func (a *AttachedClusterController) reconcile(ctx context.Context, attachedCluster *clusterv1alpha1.AttachedCluster) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	attachedCluster.Status.Ready = false

	var secret corev1.Secret
	secretKey := types.NamespacedName{Name: attachedCluster.GetSecretName(), Namespace: attachedCluster.Namespace}

	if err := a.Get(ctx, secretKey, &secret); err != nil {
		log.Error(err, "failed to get attached cluster secret")
		return ctrl.Result{}, err
	}

	attachedClusterConfig, err := clientcmd.RESTConfigFromKubeConfig(secret.Data[attachedCluster.GetSecretKey()])
	if err != nil {
		log.Error(err, "build restconfig for controlplane failed")
		return ctrl.Result{}, fmt.Errorf("build restconfig for controlplane failed %v", err)
	}
	controlPlaneKubeClient := kubeclient.NewForConfigOrDie(attachedClusterConfig)

	// The default cluster ID for Karmada is the UID of the NamespaceSystem in the cluster.
	// If this NamespaceSystem cannot be obtained, the cluster will not be ready for registration with Karmada Fleet.
	if _, err := controlPlaneKubeClient.CoreV1().Namespaces().Get(context.TODO(), metav1.NamespaceSystem, metav1.GetOptions{}); err != nil {
		log.Error(err, "failed to get attached cluster id")
		return ctrl.Result{}, err
	}
	attachedCluster.Status.Ready = true

	return ctrl.Result{}, nil
}
