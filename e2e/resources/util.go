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

package resources

import (
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreatePatchData(original, modified interface{}) ([]byte, error) {
	originalData, originalErr := json.Marshal(original)
	if originalErr != nil {
		return nil, originalErr
	}
	modifiedData, modifiedErr := json.Marshal(modified)
	if modifiedErr != nil {
		return nil, modifiedErr
	}
	patchData, createErr := jsonpatch.CreateMergePatch(originalData, modifiedData)
	if createErr != nil {
		return nil, createErr
	}
	return patchData, nil
}

func ModifiedObjectMeta(original, modified metav1.ObjectMeta) metav1.ObjectMeta {
	if modified.Labels == nil {
		modified.Labels = original.Labels
	} else {
		for k, v := range original.Labels {
			if modified.Labels[k] == "" {
				modified.Labels[k] = v
			}
		}
	}

	if modified.Annotations == nil {
		modified.Annotations = original.Annotations
	} else {
		for k, v := range original.Annotations {
			if modified.Annotations[k] == "" {
				modified.Annotations[k] = v
			}
		}
	}

	if modified.Finalizers == nil {
		modified.Finalizers = original.Finalizers
	}
	if modified.ResourceVersion == "" {
		modified.ResourceVersion = original.ResourceVersion
	}
	return modified
}
