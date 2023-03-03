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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"

	"kurator.dev/kurator/pkg/infra/aws/bucket"
	"kurator.dev/kurator/pkg/infra/service"
)

var _ service.PodIdentity = &AWSPodIdentity{}

// AWSPodIdentity is a service for creating AWS pod identity folow the guide: https://github.com/aws/amazon-eks-pod-identity-webhook/blob/master/SELF_HOSTED_SETUP.md
// It will create a S3 bucket and put the OIDC files(pubicl access) and certs(private access) to the bucket, then create a OpenID connect provider.
type AWSPodIdentity struct {
	bucketName string
	config     *aws.Config
}

func NewAWSPodIdentity(bucketName string, config *aws.Config) *AWSPodIdentity {
	return &AWSPodIdentity{
		bucketName: bucketName,
		config:     config,
	}
}

func (pi *AWSPodIdentity) Reconcile() error {
	s3client, err := bucket.NewS3Client(pi.config, pi.bucketName)
	if err != nil {
		return errors.Wrapf(err, "failed to create S3 client")
	}

	s, err := session.NewSession(pi.config)
	if err != nil {
		return errors.Wrapf(err, "failed to create AWS session")
	}

	// do not recreate certs if openid connect provider exists
	if exist, _ := pi.openIDConnectProviderExists(s); exist {
		return nil
	}

	issuerHost := fmt.Sprintf("https://%s.s3.amazonaws.com", pi.bucketName)
	files, err := bucket.S3Files(issuerHost)
	if err != nil {
		return errors.Wrapf(err, "failed to get OIDC S3 files")
	}
	for _, f := range files {
		if err := s3client.PutObject(f); err != nil {
			return errors.Wrapf(err, "failed to put object %s", f.Filename)
		}
	}

	// Create OpenID connect provider
	iamSvc := iam.New(s)
	input := &iam.CreateOpenIDConnectProviderInput{
		ClientIDList: []*string{
			aws.String("sts.amazonaws.com"),
		},
		ThumbprintList: []*string{
			aws.String("ECB2CB265649752A47EF84495ACAB7A5B348782B"), // s3已知指纹
		},
		Url: aws.String(issuerHost),
	}

	_, err = iamSvc.CreateOpenIDConnectProvider(input)
	if err != nil {
		return err
	}

	return nil
}

func (pi *AWSPodIdentity) Delete() error {
	s3client, err := bucket.NewS3Client(pi.config, pi.bucketName)
	if err != nil {
		return errors.Wrapf(err, "failed to create S3 client")
	}

	if err := s3client.DeleteBucket(); err != nil {
		return errors.Wrapf(err, "failed to clean S3 bucket")
	}

	return pi.deleteOpenIDConnectProvider()
}

func (pi *AWSPodIdentity) deleteOpenIDConnectProvider() error {
	sess, err := session.NewSession(pi.config)
	if err != nil {
		return errors.Wrapf(err, "failed to create AWS session")
	}

	stsSvc := sts.New(sess)
	out, err := stsSvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return errors.Wrapf(err, "failed to get caller identity")
	}

	iamSvc := iam.New(sess)
	_, err = iamSvc.GetOpenIDConnectProvider(&iam.GetOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: aws.String(OpenIDConnectProviderArn(*out.Account, pi.bucketName)),
	})

	if err != nil {
		if awserror, ok := err.(awserr.Error); ok {
			if awserror.Code() == iam.ErrCodeNoSuchEntityException {
				return nil
			}
		}

		return errors.Wrapf(err, "failed to get OIDC provider")
	}

	input := &iam.DeleteOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: aws.String(OpenIDConnectProviderArn(*out.Account, pi.bucketName)),
	}

	_, err = iamSvc.DeleteOpenIDConnectProvider(input)
	if err != nil {
		return errors.Wrapf(err, "failed to delete OIDC provider")
	}
	return nil
}

func (pi *AWSPodIdentity) openIDConnectProviderExists(sess *session.Session) (bool, error) {
	stsSvc := sts.New(sess)
	out, err := stsSvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get caller identity")
	}

	iamSvc := iam.New(sess)
	_, err = iamSvc.GetOpenIDConnectProvider(&iam.GetOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: aws.String(OpenIDConnectProviderArn(*out.Account, pi.bucketName)),
	})

	if err != nil {
		if awserror, ok := err.(awserr.Error); ok {
			if awserror.Code() == iam.ErrCodeNoSuchEntityException {
				return false, nil
			}
		}

		return false, errors.Wrapf(err, "failed to get OIDC provider")
	}

	return true, nil
}

func (pi *AWSPodIdentity) ServiceAccountIssuer() string {
	return fmt.Sprintf("https://%s.s3.amazonaws.com", pi.bucketName)
}
