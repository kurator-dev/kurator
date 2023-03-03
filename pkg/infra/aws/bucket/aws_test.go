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

package bucket

import (
	"bytes"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/stretchr/testify/assert"
)

// The following will use local aws config to run test
func TestS3ClientMakeBucket(t *testing.T) {
	awsConfig := &aws.Config{
		Region: aws.String("us-east-2"),
	}
	awsConfig = awsConfig.WithCredentials(credentials.NewSharedCredentials("", ""))

	bucket := "kurator-lacal-test"
	client, err := NewS3Client(awsConfig, bucket)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	exist := client.BucketExists()
	assert.Equal(t, exist, false)

	err = client.MakeBucket()
	assert.NoError(t, err)

	exist = client.BucketExists()
	assert.Equal(t, exist, true)

	err = client.DeleteBucket()
	assert.NoError(t, err)

	exist = client.BucketExists()
	assert.Equal(t, exist, false)
}

func TestS3Client(t *testing.T) {
	awsConfig := &aws.Config{
		Region: aws.String("us-east-2"),
	}
	awsConfig = awsConfig.WithCredentials(credentials.NewSharedCredentials("", ""))

	bucket := "kurator-lacal-test"
	client, err := NewS3Client(awsConfig, bucket)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	exist := client.BucketExists()
	assert.Equal(t, exist, false)

	err = client.MakeBucket()
	assert.NoError(t, err)

	err = client.PutObject(&File{
		Filename: "test.json",
		Buffer:   bytes.NewBufferString("123456"),
		ACL:      "public-read",
	})
	assert.NoError(t, err)

	exist = client.BucketExists()
	assert.Equal(t, exist, true)

	err = client.DeleteBucket()
	assert.NoError(t, err)

	exist = client.BucketExists()
	assert.Equal(t, exist, false)
}
