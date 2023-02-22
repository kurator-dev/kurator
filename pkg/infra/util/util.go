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
	"bytes"
	"fmt"
	"hash/fnv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	hashutil "github.com/karmada-io/karmada/pkg/util/hash"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrl "sigs.k8s.io/controller-runtime"

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
		return nil, errors.Wrapf(err, "failed to build resources")
	}
	if _, err := c.HelmClient().Update(target, target, true); err != nil {
		return nil, errors.Wrapf(err, "failed to update resources")
	}

	return target, nil
}

func AWSConfig(region string, credSecret *corev1.Secret) *aws.Config {
	awsConfig := &aws.Config{
		Region: aws.String(region),
	}

	accessKeyID := string(credSecret.Data["AccessKeyID"])
	secretAccessKey := string(credSecret.Data["SecretAccessKey"])
	sessionToken := string(credSecret.Data["SessionToken"])

	staticCreds := credentials.NewStaticCredentials(accessKeyID, secretAccessKey, sessionToken)
	awsConfig = awsConfig.WithCredentials(staticCreds)

	return awsConfig
}

func GenerateUID(nn types.NamespacedName) string {
	hash := fnv.New32a()
	hashutil.DeepHashObject(hash, nn.String())
	return rand.SafeEncodeString(fmt.Sprint(hash.Sum32()))
}
