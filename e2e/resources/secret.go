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

package resources

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// NewSecret will build a secret object.
func NewSecret(namespace string, name string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: data,
	}
}

// CreateSecret create or update Secret.
func CreateOrUpdateSecret(client kubernetes.Interface, secret *corev1.Secret) error {
	_, createErr := client.CoreV1().Secrets(secret.GetNamespace()).Create(context.TODO(), secret, metav1.CreateOptions{})
	if createErr != nil {
		if apierrors.IsAlreadyExists(createErr) {
			originalSecret, getErr := client.CoreV1().Secrets(secret.GetNamespace()).Get(context.TODO(), secret.GetName(), metav1.GetOptions{})
			if getErr != nil {
				return getErr
			}
			modifiedObjectMeta := ModifiedObjectMeta(originalSecret.ObjectMeta, secret.ObjectMeta)
			oldSecret := corev1.Secret{
				ObjectMeta: originalSecret.ObjectMeta,
				Data:       originalSecret.Data,
			}
			modSecret := corev1.Secret{
				ObjectMeta: modifiedObjectMeta,
				Data:       secret.Data,
			}
			secretPatchData, createPatchErr := CreatePatchData(oldSecret, modSecret)
			if createPatchErr != nil {
				return createPatchErr
			}
			_, patchErr := client.CoreV1().Secrets(secret.GetNamespace()).Patch(context.TODO(), secret.GetName(), types.StrategicMergePatchType, secretPatchData, metav1.PatchOptions{})
			if patchErr != nil {
				return patchErr
			}
		} else {
			return createErr
		}
	}
	return nil
}

func RemoveSecret(client kubernetes.Interface, namespace, name string) error {
	err := client.CoreV1().Secrets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}
	return nil
}
