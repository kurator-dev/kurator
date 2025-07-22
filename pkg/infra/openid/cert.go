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

package openid

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/square/go-jose.v2"
)

type Cert struct {
	PrivateKey []byte
	PublicKey  []byte
	KeyResponse
}

type KeyResponse struct {
	Keys []jose.JSONWebKey `json:"keys"`
}

// copied from https://github.com/aws/amazon-eks-pod-identity-webhook/blob/master/hack/self-hosted/main.go
// Refer to aws/amazon-eks-pod-identity-webhook
func NewCert() (*Cert, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	publicKey := privateKey.PublicKey

	var (
		privateKeyBuffer []byte
		publicKeyBuffer  []byte
	)

	privateKeyBuffer = x509.MarshalPKCS1PrivateKey(privateKey)
	publicKeyBuffer, err = x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return nil, err
	}

	pubKey, err := x509.ParsePKIXPublicKey(publicKeyBuffer)
	if err != nil {
		return nil, errors.Wrapf(err, "Error parsing key content")
	}
	switch pubKey.(type) {
	case *rsa.PublicKey:
	default:
		return nil, errors.New("Public key was not RSA")
	}

	var alg jose.SignatureAlgorithm
	switch pubKey.(type) {
	case *rsa.PublicKey:
		alg = jose.RS256
	default:
		return nil, fmt.Errorf("invalid public key type must be *rsa.PrivateKey")
	}

	kid, err := keyIDFromPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	keys := make([]jose.JSONWebKey, 0, 1)
	keys = append(keys, jose.JSONWebKey{
		Key:       pubKey,
		KeyID:     kid,
		Algorithm: string(alg),
		Use:       "sig",
	}, jose.JSONWebKey{
		Key:       pubKey,
		KeyID:     "",
		Algorithm: string(alg),
		Use:       "sig",
	})

	return &Cert{
		// return PEM encoded keys
		PrivateKey: pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBuffer,
		}),
		// return PEM encoded keys
		PublicKey: pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBuffer,
		}),
		KeyResponse: KeyResponse{
			Keys: keys,
		},
	}, nil
}

// copied from kubernetes/kubernetes#78502
func keyIDFromPublicKey(publicKey interface{}) (string, error) {
	publicKeyDERBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to serialize public key to DER format: %v", err)
	}

	hasher := crypto.SHA256.New()
	hasher.Write(publicKeyDERBytes)
	publicKeyDERHash := hasher.Sum(nil)

	keyID := base64.RawURLEncoding.EncodeToString(publicKeyDERHash)

	return keyID, nil
}
