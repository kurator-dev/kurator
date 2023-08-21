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

package clusteroperator

import (
	"context"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"kurator.dev/kurator/cmd/cluster-operator/scheme"
	"kurator.dev/kurator/pkg/apis/infra/v1alpha1"
)

func generatePodOwnerRefCluster(clusterName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-pod",
			Finalizers: []string{
				"customcluster.cluster.kurator.dev",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: CustomClusterKind,
					Name: clusterName,
					UID:  types.UID(clusterName),
				},
			},
		},
	}
}

func generateCustomMachineOwnerRefCustomCluster(clusterName string) *v1alpha1.CustomMachine {
	return &v1alpha1.CustomMachine{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "test-machine",
			Finalizers: []string{
				"customcluster.cluster.kurator.dev",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       CustomClusterKind,
					Name:       clusterName,
					UID:        types.UID(clusterName),
				},
			},
		},
		Spec: v1alpha1.CustomMachineSpec{
			Master: []v1alpha1.Machine{
				{
					SSHKey: &corev1.ObjectReference{
						Kind:      "customMachine",
						Namespace: "test",
						Name:      clusterName,
					},
					PublicIP: "172.19.3.75",
				},
			},
		},
	}
}

func generateCustomCluster(customClusterName string) *v1alpha1.CustomCluster {
	return &v1alpha1.CustomCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      customClusterName,
			UID:       types.UID(customClusterName),
			Finalizers: []string{
				"customcluster.cluster.kurator.dev",
			},
		},
	}
}

func generateCluster(clusterName string) *clusterv1.Cluster {
	return &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      clusterName,
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       CustomClusterKind,
				Namespace:  "test",
				Name:       clusterName,
			},
			ClusterNetwork: &clusterv1.ClusterNetwork{
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{
						"172.19.3.75",
					},
				},
				Services: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{
						"172.18.0.3",
					},
				},
				ServiceDomain: "",
			},
		},
	}
}

func generateKcp(kcpName string) *controlplanev1.KubeadmControlPlane {
	return &controlplanev1.KubeadmControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      kcpName,
			Finalizers: []string{
				"customcluster.cluster.kurator.dev",
			},
		},
		Spec: controlplanev1.KubeadmControlPlaneSpec{
			Version: "v1",
			KubeadmConfigSpec: bootstrapv1.KubeadmConfigSpec{
				ClusterConfiguration: &bootstrapv1.ClusterConfiguration{
					ClusterName: "customCluster",
				},
			},
		},
	}
}

func TestCustomClusterController_deleteWorkerPods(t *testing.T) {
	testCustomCluster := generateCustomCluster("customcluster")
	testKcp := generateKcp("testKcp")
	// init worker pod
	workerPod1 := generateClusterManageWorker(testCustomCluster, CustomClusterInitAction, KubesprayInitCMD,
		generateClusterHostsName(testCustomCluster), generateClusterHostsName(testCustomCluster), testKcp.Spec.Version)
	workerPod1.ObjectMeta.SetOwnerReferences([]metav1.OwnerReference{
		{Name: "customcluster",
			UID: types.UID("customcluster"),
		},
	})
	// scale down
	workerPod2 := generateClusterManageWorker(testCustomCluster, CustomClusterScaleDownAction, KubesprayScaleDownCMDPrefix,
		generateClusterHostsName(testCustomCluster), generateClusterHostsName(testCustomCluster), testKcp.Spec.Version)
	workerPod2.ObjectMeta.SetOwnerReferences([]metav1.OwnerReference{
		{Name: "customcluster",
			UID: types.UID("customcluster"),
		},
	})
	// scale up
	workerPod3 := generateClusterManageWorker(testCustomCluster, CustomClusterScaleUpAction, KubesprayScaleUpCMD,
		generateClusterHostsName(testCustomCluster), generateClusterHostsName(testCustomCluster), testKcp.Spec.Version)
	workerPod3.ObjectMeta.SetOwnerReferences([]metav1.OwnerReference{
		{Name: "customcluster",
			UID: types.UID("customcluster"),
		},
	})

	type fields struct {
		Client    client.Client
		APIReader client.Reader
		Scheme    *runtime.Scheme
	}
	type args struct {
		ctx           context.Context
		customCluster *v1alpha1.CustomCluster
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "delete the init work pod",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(testCustomCluster, workerPod1).Build(),
			},
			args: args{
				ctx:           context.Background(),
				customCluster: testCustomCluster,
			},
			wantErr: false,
		},
		{
			name: "delete the scale down work pod",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(testCustomCluster, workerPod2).Build(),
			},
			args: args{
				ctx:           context.Background(),
				customCluster: testCustomCluster,
			},
			wantErr: false,
		},
		{
			name: "delete the scale up work pod",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(testCustomCluster, workerPod3).Build(),
			},
			args: args{
				ctx:           context.Background(),
				customCluster: testCustomCluster,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CustomClusterController{
				Client:    tt.fields.Client,
				APIReader: tt.fields.APIReader,
				Scheme:    tt.fields.Scheme,
			}
			err1 := r.deleteWorkerPods(tt.args.ctx, tt.args.customCluster)
			assert.Empty(t, err1)

			patches := gomonkey.ApplyPrivateMethod(reflect.TypeOf(r), "ensureWorkerPodDeleted",
				func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction) error {
					err := errors.New("failed to delete Pod")
					return err
				})
			defer patches.Reset()
			err2 := r.deleteWorkerPods(tt.args.ctx, tt.args.customCluster)
			assert.NotEmpty(t, err2)
		})
	}
}

func TestCustomClusterController_CustomMachineToCustomClusterMapFunc(t *testing.T) {
	testPod := generatePodOwnerRefCluster("testCluster")
	testCustomMachine := generateCustomMachineOwnerRefCustomCluster("testCustomCluster")
	testCustomCluster := generateCustomCluster("testCustomCluster")

	type fields struct {
		Client    client.Client
		APIReader client.Reader
		Scheme    *runtime.Scheme
	}
	type args struct {
		o client.Object
	}
	tests := []struct {
		name      string
		wantError bool
		fields    fields
		args      args
	}{
		{
			name:      "not custommachine error",
			wantError: true,
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(testPod, testCustomMachine, testCustomCluster).Build(),
			},
			args: args{
				o: testPod,
			},
		},
		{
			name:      "custommachine test",
			wantError: false,
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(testPod, testCustomMachine, testCustomCluster).Build(),
			},
			args: args{
				o: testCustomMachine,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CustomClusterController{
				Client:    tt.fields.Client,
				APIReader: tt.fields.APIReader,
				Scheme:    tt.fields.Scheme,
			}

			defer func() {
				err := recover()
				if err == nil && tt.wantError {
					t.Errorf("this code did not panic %v", err)
				}
			}()
			actual := r.CustomMachineToCustomClusterMapFunc(tt.args.o)
			expect := []ctrl.Request{{NamespacedName: client.ObjectKey{Namespace: "test", Name: "testCustomCluster"}}}
			assert.Equal(t, expect, actual)
		})
	}
}

func TestCustomClusterController_WorkerToCustomClusterMapFunc(t *testing.T) {
	testPod := generatePodOwnerRefCluster("testCluster")
	testCustomMachine := generateCustomMachineOwnerRefCustomCluster("testCustomCluster")
	testCustomCluster := generateCustomCluster("testCustomCluster")

	type fields struct {
		Client    client.Client
		APIReader client.Reader
		Scheme    *runtime.Scheme
	}
	type args struct {
		o client.Object
	}
	tests := []struct {
		name      string
		wantError bool
		fields    fields
		args      args
	}{
		{
			name:      "not pod error",
			wantError: true,
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(testPod, testCustomMachine, testCustomCluster).Build(),
			},
			args: args{
				o: testCustomMachine,
			},
		},
		{
			name:      "pod test",
			wantError: false,
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(testPod, testCustomMachine, testCustomCluster).Build(),
			},
			args: args{
				o: testPod,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CustomClusterController{
				Client:    tt.fields.Client,
				APIReader: tt.fields.APIReader,
				Scheme:    tt.fields.Scheme,
			}
			defer func() {
				err := recover()
				if err == nil && tt.wantError {
					t.Errorf("this code did not panic %v", err)
				}
			}()
			actual := r.WorkerToCustomClusterMapFunc(tt.args.o)
			expect := []ctrl.Request{{NamespacedName: client.ObjectKey{Namespace: "default", Name: "testCluster"}}}
			assert.Equal(t, expect, actual)
		})
	}
}

func TestCustomClusterController_ClusterToCustomClusterMapFunc(t *testing.T) {
	testPod := generatePodOwnerRefCluster("testCluster")
	testCluster := generateCluster("testCluster")
	testCustomCluster := generateCustomCluster("testCustomCluster")

	type fields struct {
		Client    client.Client
		APIReader client.Reader
		Scheme    *runtime.Scheme
	}
	type args struct {
		o client.Object
	}
	tests := []struct {
		name      string
		wantError bool
		fields    fields
		args      args
	}{
		{
			name:      "not cluster error",
			wantError: true,
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(testPod, testCluster, testCustomCluster).Build(),
			},
			args: args{
				o: testPod,
			},
		},
		{
			name:      "cluster test",
			wantError: false,
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
					WithObjects(testPod, testCluster, testCustomCluster).Build(),
			},
			args: args{
				o: testCluster,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CustomClusterController{
				Client:    tt.fields.Client,
				APIReader: tt.fields.APIReader,
				Scheme:    tt.fields.Scheme,
			}
			defer func() {
				err := recover()
				if err == nil && tt.wantError {
					t.Errorf("this code did not panic %v", err)
				}
			}()
			actual := r.ClusterToCustomClusterMapFunc(tt.args.o)
			expect := []ctrl.Request{{NamespacedName: client.ObjectKey{Namespace: "test", Name: "testCluster"}}}
			assert.Equal(t, expect, actual)
		})
	}
}

func TestCustomClusterController_ensureFinalizerAndOwnerRef(t *testing.T) {
	ctx := context.Background()
	testCustomCluster := generateCustomCluster("customCluster")
	testCustomMachine := generateCustomMachineOwnerRefCustomCluster("customCluster")
	testKcp := generateKcp("kcp")
	testClusterHosts := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "test-clusterhost",
		},
	}
	testClusterConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "test-clusterconfig",
		},
	}

	r := &CustomClusterController{
		Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
			WithRuntimeObjects(testCustomCluster, testCustomMachine, testKcp, testClusterConfig, testClusterHosts).
			Build(),
	}
	err := r.ensureFinalizerAndOwnerRef(ctx, testClusterHosts, testClusterConfig, testCustomCluster, testCustomMachine, testKcp)
	if err != nil {
		t.Errorf("customcluster_controller ensureFinalizerAndOwnerRef() error is %v", err)
		return
	}
}

func TestCustomClusterController_deleteResource(t *testing.T) {
	ctx := context.Background()
	testCustomCluster := generateCustomCluster("customCluster")
	testCustomMachine := generateCustomMachineOwnerRefCustomCluster("customCluster")
	testKcp := generateKcp("kcp")
	r := &CustomClusterController{
		Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
			WithObjects(testCustomCluster, testCustomMachine, testKcp).Build(),
	}
	patches := gomonkey.NewPatches()

	testCases := []struct {
		name       string
		wantError  bool
		beforeFunc func()
		afterFunc  func()
	}{
		{
			name:      "delete the configmap cluster-config failed",
			wantError: true,
			beforeFunc: func() {
				patches.ApplyPrivateMethod(reflect.TypeOf(r), "ensureConfigMapDeleted",
					func(_ *CustomClusterController, ctx context.Context, cmKey types.NamespacedName) error {
						return errors.New("failed to ensure that configmap is deleted")
					})
			},
			afterFunc: func() {
				patches.Reset()
			},
		},
		{
			name:       "no error test",
			wantError:  false,
			beforeFunc: func() {},
			afterFunc:  func() {},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tt.beforeFunc()
			err := r.deleteResource(ctx, testCustomCluster, testCustomMachine, testKcp)
			if err != nil && !tt.wantError {
				t.Errorf("%v", err)
			}
			tt.afterFunc()
		})
	}
}

func TestCustomClusterController_reconcileDelete(t *testing.T) {
	ctx := context.Background()
	testCustomCluster := generateCustomCluster("customCluster")
	testCustomMachine := generateCustomMachineOwnerRefCustomCluster("customCluster")
	testKcp := generateKcp("kcp")
	r := &CustomClusterController{
		Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
			WithObjects(testCustomCluster, testCustomMachine, testKcp).Build(),
	}
	patches1 := gomonkey.NewPatches()
	patches2 := gomonkey.NewPatches()

	testCases := []struct {
		name       string
		wantError  bool
		beforeFunc func()
		afterFunc  func()
	}{
		{
			name:      "failed to delete worker pods",
			wantError: true,
			beforeFunc: func() {
				patches1.ApplyPrivateMethod(reflect.TypeOf(r), "deleteWorkerPods",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster) error {
						return errors.New("failed to deleted worker pods")
					})
			},
			afterFunc: func() {
				patches1.Reset()
			},
		},
		{
			name:      "failed to create terminate worker",
			wantError: true,
			beforeFunc: func() {
				patches1.ApplyPrivateMethod(reflect.TypeOf(r), "ensureWorkerPodCreated",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction,
						manageCMD customClusterManageCMD, hostName, configName, kubeVersion string) (*corev1.Pod, error) {
						workerPod := generatePodOwnerRefCluster("pod")
						return workerPod, errors.New("failed to create terminate worker pods")
					})
			},
			afterFunc: func() {
				patches1.Reset()
			},
		},
		{
			name:      "cluster delete cluster but delete CRD failed",
			wantError: true,
			beforeFunc: func() {
				patches1.ApplyPrivateMethod(reflect.TypeOf(r), "ensureWorkerPodCreated",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction,
						manageCMD customClusterManageCMD, hostName, configName, kubeVersion string) (*corev1.Pod, error) {
						workerPod := generatePodOwnerRefCluster("pod")
						workerPod.Status.Phase = corev1.PodSucceeded
						return workerPod, nil
					})
				patches2.ApplyPrivateMethod(reflect.TypeOf(r), "deleteResource",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, kcp *controlplanev1.KubeadmControlPlane) error {
						return errors.New("failed to delete worker pods")
					})
			},
			afterFunc: func() {
				patches1.Reset()
				patches2.Reset()
			},
		},
		{
			name:      "termination worker pod create successful but run failed",
			wantError: false,
			beforeFunc: func() {
				patches1.ApplyPrivateMethod(reflect.TypeOf(r), "ensureWorkerPodCreated",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction,
						manageCMD customClusterManageCMD, hostName, configName, kubeVersion string) (*corev1.Pod, error) {
						workerPod := generatePodOwnerRefCluster("pod")
						workerPod.Status.Phase = corev1.PodFailed
						return workerPod, nil
					})
			},
			afterFunc: func() {
				patches1.Reset()
			},
		},
		{
			name:       "no error test",
			wantError:  false,
			beforeFunc: func() {},
			afterFunc:  func() {},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tt.beforeFunc()
			_, err := r.reconcileDelete(ctx, testCustomCluster, testCustomMachine, testKcp)
			if err != nil && !tt.wantError {
				t.Errorf("%v", err)
			}
			tt.afterFunc()
		})
	}
}

func TestCustomClusterController_reconcileProvision(t *testing.T) {
	ctx := context.Background()
	testCustomCluster := generateCustomCluster("customCluster")
	testCustomMachine := generateCustomMachineOwnerRefCustomCluster("customCluster")
	testCluster := generateCluster("cluster")
	testKcp := generateKcp("kcp")
	r := &CustomClusterController{
		Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).
			WithObjects(testCustomCluster, testCustomMachine, testKcp).Build(),
	}

	patches1 := gomonkey.NewPatches()
	patches2 := gomonkey.NewPatches()
	testCases := []struct {
		name       string
		wantError  bool
		beforeFunc func()
		afterFunc  func()
	}{
		{
			name:      "failed to update cluster-hosts configmap",
			wantError: true,
			beforeFunc: func() {
				patches1.ApplyPrivateMethod(reflect.TypeOf(r), "ensureClusterHostsCreated",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, kcp *controlplanev1.KubeadmControlPlane) (*corev1.ConfigMap, error) {
						emptyCM := &corev1.ConfigMap{}
						return emptyCM, errors.New("failed to update cluster-hosts configmap")
					})
			},
			afterFunc: func() {
				patches1.Reset()
			},
		},
		{
			name:      "failed to update cluster-config configmap",
			wantError: true,
			beforeFunc: func() {
				patches1.ApplyPrivateMethod(reflect.TypeOf(r), "ensureClusterConfigCreated",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, kcp *controlplanev1.KubeadmControlPlane) (*corev1.ConfigMap, error) {
						emptyCM := &corev1.ConfigMap{}
						return emptyCM, errors.New("failed to update cluster-config configmap")
					})
			},
			afterFunc: func() {
				patches1.Reset()
			},
		},
		{
			name:      "init worker is failed to create",
			wantError: true,
			beforeFunc: func() {
				patches1.ApplyPrivateMethod(reflect.TypeOf(r), "ensureWorkerPodCreated",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction,
						manageCMD customClusterManageCMD, hostName, configName, kubeVersion string) (*corev1.Pod, error) {
						workerPod := generatePodOwnerRefCluster("pod")
						return workerPod, errors.New("init worker is failed to create")
					})
			},
			afterFunc: func() {
				patches1.Reset()
			},
		},
		{
			name:      "failed to set finalizer or ownerRefs",
			wantError: true,
			beforeFunc: func() {
				patches1.ApplyPrivateMethod(reflect.TypeOf(r), "ensureFinalizerAndOwnerRef",
					func(_ *CustomClusterController, ctx context.Context, clusterHosts *corev1.ConfigMap, clusterConfig *corev1.ConfigMap, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine, kcp *controlplanev1.KubeadmControlPlane) error {
						return errors.New("failed to set finalizer or ownerRefs")
					})
			},
			afterFunc: func() {
				patches1.Reset()
			},
		},
		{
			name:      "init worker create successful but run failed",
			wantError: false,
			beforeFunc: func() {
				patches1.ApplyPrivateMethod(reflect.TypeOf(r), "ensureWorkerPodCreated",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction,
						manageCMD customClusterManageCMD, hostName, configName, kubeVersion string) (*corev1.Pod, error) {
						workerPod := generatePodOwnerRefCluster("pod")
						workerPod.Status.Phase = corev1.PodFailed
						return workerPod, nil
					})
			},
			afterFunc: func() {
				patches1.Reset()
			},
		},
		{
			name:      "init worker finished successful but failed to fetch provisioned cluster kubeConfig",
			wantError: true,
			beforeFunc: func() {
				patches1.ApplyPrivateMethod(reflect.TypeOf(r), "ensureWorkerPodCreated",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction,
						manageCMD customClusterManageCMD, hostName, configName, kubeVersion string) (*corev1.Pod, error) {
						workerPod := generatePodOwnerRefCluster("pod")
						workerPod.Status.Phase = corev1.PodSucceeded
						return workerPod, nil
					})

				patches2.ApplyPrivateMethod(reflect.TypeOf(r), "fetchProvisionedClusterKubeConfig",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine) error {
						return errors.New("failed to fetch provisioned cluster kubeConfig")
					})
			},
			afterFunc: func() {
				patches1.Reset()
				patches2.Reset()
			},
		},
		{
			name:      "init worker finished successful",
			wantError: false,
			beforeFunc: func() {
				patches1.ApplyPrivateMethod(reflect.TypeOf(r), "ensureWorkerPodCreated",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, manageAction customClusterManageAction,
						manageCMD customClusterManageCMD, hostName, configName, kubeVersion string) (*corev1.Pod, error) {
						workerPod := generatePodOwnerRefCluster("pod")
						workerPod.Status.Phase = corev1.PodSucceeded
						return workerPod, nil
					})
				patches2.ApplyPrivateMethod(reflect.TypeOf(r), "fetchProvisionedClusterKubeConfig",
					func(_ *CustomClusterController, ctx context.Context, customCluster *v1alpha1.CustomCluster, customMachine *v1alpha1.CustomMachine) error {
						return nil
					})
			},
			afterFunc: func() {
				patches1.Reset()
				patches2.Reset()
			},
		},
		{
			name:       "no error test",
			wantError:  false,
			beforeFunc: func() {},
			afterFunc:  func() {},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tt.beforeFunc()
			_, err := r.reconcileProvision(ctx, testCustomCluster, testCustomMachine, testCluster, testKcp)
			if err != nil && !tt.wantError {
				t.Errorf("%v", err)
			}
			tt.afterFunc()
		})
	}
}
