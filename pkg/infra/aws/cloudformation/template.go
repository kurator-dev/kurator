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

package cloudformation

import (
	"fmt"

	cf "github.com/awslabs/goformation/v4/cloudformation"
	capabootstrap "sigs.k8s.io/cluster-api-provider-aws/v2/cmd/clusterawsadm/cloudformation/bootstrap"
	iamv1 "sigs.k8s.io/cluster-api-provider-aws/v2/iam/api/v1beta1"

	"kurator.dev/kurator/pkg/infra/scope"
)

// Template returns the cloudformation template for the cluster.
// If cluster with pod identity enabled, it will add policy to get key cert from specific s3 bucket.
func Template(cluster *scope.Cluster) *cf.Template {
	iamTpl := capabootstrap.NewTemplate()
	s := cluster.StackSuffix()
	iamTpl.Spec.NameSuffix = &s
	iamTpl.Spec.EKS.Disable = true
	if cluster.EnablePodIdentity {
		iamTpl.Spec.StackName = fmt.Sprintf("%s%s", "pod-identity", s) // use to attack policy to role
		// add policy for get key cert from s3
		iamTpl.Spec.ControlPlane.ExtraStatements = append(iamTpl.Spec.ClusterAPIControllers.ExtraStatements, iamv1.StatementEntry{
			Effect:   iamv1.EffectAllow,
			Action:   []string{"s3:GetObject"},
			Resource: []string{fmt.Sprintf("arn:*:s3:::%s/*", cluster.BucketName)},
		})
	}

	return iamTpl.RenderCloudFormation()
}
