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

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/stretchr/testify/assert"
)

// The following will use local aws config to run test
func TestAWSPodIdentity(t *testing.T) {
	awsConfig := &aws.Config{
		Region: aws.String("us-east-2"),
	}
	awsConfig = awsConfig.WithCredentials(credentials.NewSharedCredentials("", ""))

	prov := NewAWSPodIdentity("kurator-local-test", awsConfig)

	err := prov.Reconcile()
	assert.NoError(t, err)

	err = prov.Delete()
	assert.NoError(t, err)

	// TODO: find a way to verify OpenID connect provider work as expected
}
