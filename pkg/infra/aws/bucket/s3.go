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
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"

	"kurator.dev/kurator/pkg/infra/openid"
)

type File struct {
	Filename string
	Buffer   *bytes.Buffer
	ACL      string
}

type Client interface {
	BucketExists() bool
	MakeBucket() error
	PutObject(f *File) error
	DeleteBucket() error
}

var _ Client = &s3Client{}

type s3Client struct {
	bucketName string
	sess       *session.Session
}

func NewS3Client(awsCfg *aws.Config, bucketName string) (Client, error) {
	s, err := session.NewSession(awsCfg)
	if err != nil {
		return nil, err
	}
	return &s3Client{
		sess:       s,
		bucketName: bucketName,
	}, nil
}

func (c *s3Client) BucketExists() bool {
	svc := s3.New(c.sess)
	_, err := svc.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(c.bucketName),
	})
	return err == nil
}

func (c *s3Client) MakeBucket() error {
	svc := s3.New(c.sess)
	_, err := svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(c.bucketName),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create bucket %s", c.bucketName)
	}
	return nil
}

func (c *s3Client) PutObject(f *File) error {
	if !c.BucketExists() {
		if err := c.MakeBucket(); err != nil {
			return fmt.Errorf("failed to create bucket %s, %v", c.bucketName, err)
		}
	}

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(c.sess)
	_, err := uploader.Upload(&s3manager.UploadInput{
		ACL:    aws.String(f.ACL),
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(f.Filename),
		Body:   f.Buffer,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}

	return nil
}

func (c *s3Client) DeleteBucket() error {
	if exists := c.BucketExists(); !exists {
		return nil
	}

	svc := s3.New(c.sess)
	if err := c.cleanBucket(svc); err != nil {
		return err
	}

	_, err := svc.DeleteBucket(&s3.DeleteBucketInput{Bucket: aws.String(c.bucketName)})
	if err != nil {
		return errors.Wrapf(err, "failed to delete bucket %s", c.bucketName)
	}

	return nil
}

func (c *s3Client) cleanBucket(svc *s3.S3) error {
	// Setup BatchDeleteIterator to iterate through a list of objects.
	iter := s3manager.NewDeleteListIterator(svc, &s3.ListObjectsInput{
		Bucket: aws.String(c.bucketName),
	})

	// Traverse iterator deleting each object
	if err := s3manager.NewBatchDeleteWithClient(svc).Delete(aws.BackgroundContext(), iter); err != nil {
		return errors.Wrapf(err, "failed to delete object in bucket %s", c.bucketName)
	}

	return nil
}

const (
	PublicReadACL = "public-read"
	PrivateACL    = "private"
)

func S3Files(issuerHost string) ([]*File, error) {
	cert, err := openid.NewCert()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create oidc cert")
	}

	files := make([]*File, 0, 4)
	// openid-configuration
	discovery := openid.OpenIDConfiguration(issuerHost)
	files = append(files, &File{
		Filename: ".well-known/openid-configuration",
		Buffer:   bytes.NewBufferString(discovery),
		ACL:      PublicReadACL,
	})

	// upload keys.json
	keysJSONBuff, err := json.Marshal(cert.KeyResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keys.json, %v", err)
	}
	files = append(files, &File{
		Filename: "keys.json",
		Buffer:   bytes.NewBuffer(keysJSONBuff),
		ACL:      PublicReadACL,
	})

	// sa-signer.key
	files = append(files, &File{
		Filename: "sa-signer.key",
		Buffer:   bytes.NewBuffer(cert.PrivateKey),
		ACL:      PrivateACL,
	})

	// sa-signer-pkcs8.pub
	files = append(files, &File{
		Filename: "sa-signer-pkcs8.pub",
		Buffer:   bytes.NewBuffer(cert.PublicKey),
		ACL:      PrivateACL,
	})

	return files, nil
}
