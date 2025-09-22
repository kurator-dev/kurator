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

package fleet

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

const ServiceAccountName = "fleet-manager-worker"
const KarmadaCtlImage = "ghcr.io/kurator-dev/karmadactl:v0.1.0"
const FleetWorkerClusterRoleBindingName = "fleet-worker"

func (f *FleetManager) reconcileControlPlane(ctx context.Context, fleet *fleetapi.Fleet) error {
	controlplane := fleet.Annotations[fleetapi.ControlplaneAnnotation]
	// if no controlplane is specified, do nothing
	// we do not support annotation update yet
	if controlplane == "" {
		fleet.Status.Phase = fleetapi.ReadyPhase
		fleet.Status.CredentialSecret = nil
		return nil
	}
	// TODO: generate a valid name
	podName := fleet.Name + "-init"
	namespace := fleet.Namespace

	clusterKey := types.NamespacedName{Name: podName, Namespace: namespace}
	var pod corev1.Pod
	err := f.Get(ctx, clusterKey, &pod)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		// pod already exists
		if pod.Status.Phase == corev1.PodSucceeded {
			// pod is done, update the fleet status
			fleet.Status.Phase = fleetapi.ReadyPhase
			secret := "kubeconfig"
			// TODO: update the kubeconfig api endpoint?
			// "kubeconfig" is the name of the kubeconfig to access karmada apiserver.
			fleet.Status.CredentialSecret = &secret
		}
		return nil
	}

	ownerref := ownerReference(fleet)
	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      ServiceAccountName,
			Labels: map[string]string{
				FleetLabel: fleet.Name,
			},
			OwnerReferences: []metav1.OwnerReference{*ownerref},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
	}

	if err = f.Create(ctx, &sa); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			ctrl.LoggerFrom(ctx).Error(err, "unable to create sa", "pod", types.NamespacedName{Name: podName, Namespace: namespace})
			return fmt.Errorf("failed to create sa for init pod: %v", err)
		}
	}

	// update rolebinding
	var clusterRolebinding rbacv1.ClusterRoleBinding
	key := types.NamespacedName{Name: FleetWorkerClusterRoleBindingName}
	if err = f.Get(ctx, key, &clusterRolebinding); err != nil {
		return fmt.Errorf("failed to get clusterrolebinding for init pod: %v", err)
	}
	clusterRolebinding.Subjects = append(clusterRolebinding.Subjects, rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      ServiceAccountName,
		Namespace: namespace,
	})
	if err = f.Update(ctx, &clusterRolebinding); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to update clusterrolebinding", "pod", types.NamespacedName{Name: podName, Namespace: namespace})
		return fmt.Errorf("failed to update clusterrolebinding for init pod: %v", err)
	}

	// pod not found, create it
	initCmd := "karmadactl init -n " + namespace
	pod = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      podName,
			Labels: map[string]string{
				FleetLabel: fleet.Name,
			},
			OwnerReferences: []metav1.OwnerReference{*ownerref},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    podName,
					Image:   KarmadaCtlImage,
					Command: []string{"/bin/sh", "-c"},
					Args:    []string{string(initCmd)},
				},
			},
			ServiceAccountName: ServiceAccountName,
			RestartPolicy:      corev1.RestartPolicyNever,
		},
	}

	if err = f.Create(ctx, &pod); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			ctrl.LoggerFrom(ctx).Error(err, "unable to create pod", "pod", types.NamespacedName{Name: podName, Namespace: namespace})
			return fmt.Errorf("failed to create fleet control plane init pod: %v", err)
		}
	}

	fleet.Status.Phase = fleetapi.RunningPhase
	return nil
}

func (f *FleetManager) deleteControlPlane(ctx context.Context, fleet *fleetapi.Fleet) error {
	controlplane := fleet.Annotations[fleetapi.ControlplaneAnnotation]
	// if no controlplane is specified, do nothing
	if controlplane == "" {
		fleet.Status.Phase = fleetapi.TerminateSucceededPhase
		fleet.Status.CredentialSecret = nil
		return nil
	}
	podName := fleet.Name + "-delete"
	namespace := fleet.Namespace

	clusterKey := types.NamespacedName{Name: podName, Namespace: namespace}
	var pod corev1.Pod
	err := f.Get(ctx, clusterKey, &pod)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		// pod already exists
		if pod.Status.Phase == corev1.PodSucceeded {
			// update rolebinding
			var clusterRolebinding rbacv1.ClusterRoleBinding
			key := types.NamespacedName{Name: FleetWorkerClusterRoleBindingName}
			if err = f.Get(ctx, key, &clusterRolebinding); err != nil {
				return fmt.Errorf("failed to get clusterrolebinding for init pod: %v", err)
			}
			for i, subject := range clusterRolebinding.Subjects {
				if subject.Namespace == namespace {
					clusterRolebinding.Subjects = append(clusterRolebinding.Subjects[:i], clusterRolebinding.Subjects[i+1:]...)
					break
				}
			}
			if err = f.Update(ctx, &clusterRolebinding); err != nil {
				ctrl.LoggerFrom(ctx).Error(err, "unable to update clusterrolebinding", "pod", types.NamespacedName{Name: podName, Namespace: namespace})
				return fmt.Errorf("failed to update clusterrolebinding: %v", err)
			}

			fleet.Status.Phase = fleetapi.TerminateSucceededPhase
			return nil
		}
		if pod.Status.Phase == corev1.PodFailed {
			fleet.Status.Phase = fleetapi.TerminateFailedPhase
			ctrl.LoggerFrom(ctx).Info("pod failed", "pod", types.NamespacedName{Name: podName, Namespace: namespace})
			return nil
		}
		return nil
	}

	ownerref := ownerReference(fleet)
	// pod not found, create it
	initCmd := "echo y | karmadactl deinit -n " + namespace
	pod = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      podName,
			Labels: map[string]string{
				FleetLabel: fleet.Name,
			},
			OwnerReferences: []metav1.OwnerReference{*ownerref},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},

		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    podName,
					Image:   KarmadaCtlImage,
					Command: []string{"/bin/sh", "-c"},
					Args:    []string{string(initCmd)},
				},
			},
			ServiceAccountName: ServiceAccountName,
			RestartPolicy:      corev1.RestartPolicyNever,
		},
	}

	if err = f.Create(ctx, &pod); err != nil {
		ctrl.LoggerFrom(ctx).Error(err, "unable to create pod", "pod", types.NamespacedName{Name: podName, Namespace: namespace})
		return fmt.Errorf("failed to create fleet control plane init pod: %v", err)
	}

	fleet.Status.Phase = fleetapi.TerminatingPhase
	return nil
}
