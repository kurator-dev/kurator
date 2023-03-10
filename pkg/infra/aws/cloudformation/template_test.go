//go:build aws
// +build aws

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

package cloudformation

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awssdkcfn "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	cfsvc "sigs.k8s.io/cluster-api-provider-aws/v2/cmd/clusterawsadm/cloudformation/service"

	"kurator.dev/kurator/pkg/infra/scope"
)

// The following will use local aws config to run test
func TestCloudFormationTemplate(t *testing.T) {
	awsConfig := &aws.Config{
		Region: aws.String("us-east-2"),
	}
	awsConfig = awsConfig.WithCredentials(credentials.NewSharedCredentials("", ""))
	s, err := session.NewSession(awsConfig)
	assert.NoError(t, err)
	cluster := &scope.Cluster{
		InfraType: "aws",
		UID:       "xxxxx",
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: "default",
		},
		CNIType: "calico",
	}
	tpl := Template(cluster)

	cfnSvc := cfsvc.NewService(awssdkcfn.New(s))
	err = cfnSvc.ReconcileBootstrapStack(cluster.StackName(), *tpl, nil)
	assert.NoError(t, err)
}
