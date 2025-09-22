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

package util

import (
	"bytes"
	_ "embed"
	"fmt"
	"hash/fnv"

	hashutil "github.com/karmada-io/karmada/pkg/util/hash"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	addonsv1 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"

	infrav1 "kurator.dev/kurator/pkg/apis/cluster/v1alpha1"
	"kurator.dev/kurator/pkg/client"
)

func PatchResources(b []byte) (kube.ResourceList, error) {
	rest, err := ctrl.GetConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get kubeconfig")
	}
	c, err := client.NewClient(client.NewRESTClientGetter(rest))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create client")
	}
	target, err := c.HelmClient().Build(bytes.NewBuffer(b), false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build resources: %s", string(b))
	}
	if _, err := c.HelmClient().Update(target, target, true); err != nil {
		return nil, errors.Wrapf(err, "failed to update resources")
	}

	return target, nil
}

func GenerateUID(nn types.NamespacedName) string {
	hash := fnv.New32a()
	hashutil.DeepHashObject(hash, nn.String())
	return rand.SafeEncodeString(fmt.Sprint(hash.Sum32()))
}

func AdditionalResources(infraCluster *infrav1.Cluster) []addonsv1.ResourceRef {
	refs := make([]addonsv1.ResourceRef, 0, len(infraCluster.Spec.AdditionalResources))
	for _, resource := range infraCluster.Spec.AdditionalResources {
		refs = append(refs, addonsv1.ResourceRef{
			Kind: resource.Kind,
			Name: resource.Name,
		})
	}

	return refs
}
