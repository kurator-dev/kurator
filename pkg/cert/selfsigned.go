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

package cert

import (
	"fmt"
	"time"

	k8ssecret "istio.io/istio/security/pkg/k8s/secret"
	"istio.io/istio/security/pkg/pki/ca"
	"istio.io/istio/security/pkg/pki/util"
	v1 "k8s.io/api/core/v1"

	"kurator.dev/kurator/pkg/typemeta"
)

const (
	defaultSelfSignedCACertTTL = 3650 * 24 * time.Hour
	defaultRSAKeySize          = 2048

	istioCASecretType = "istio.io/ca-root"
)

type SelfSignedCert struct {
	org string
}

func NewSelfSignedCert(org string) *SelfSignedCert {
	return &SelfSignedCert{
		org: org,
	}
}

func (cert *SelfSignedCert) gen() (*util.KeyCertBundle, error) {
	options := util.CertOptions{
		TTL:          defaultSelfSignedCACertTTL,
		Org:          cert.org,
		IsCA:         true,
		IsSelfSigned: true,
		RSAKeySize:   defaultRSAKeySize,
		IsDualUse:    true,
	}

	pemCert, pemKey, ckErr := util.GenCertKeyFromOptions(options)
	if ckErr != nil {
		return nil, fmt.Errorf("unable to generate CA cert and key for self-signed CA (%v)", ckErr)
	}

	rootCerts, err := util.AppendRootCerts(pemCert, "")
	if err != nil {
		return nil, fmt.Errorf("failed to append root certificates (%v)", err)
	}

	return util.NewVerifiedKeyCertBundleFromPem(pemCert, pemKey, nil, rootCerts)
}

func (cert *SelfSignedCert) Secret(namespace string) (*v1.Secret, error) {
	keyCertBundle, err := cert.gen()
	if err != nil {
		return nil, err
	}

	certBytes, privKeyBytes, _, _ := keyCertBundle.GetAllPem()

	secret := k8ssecret.BuildSecret(ca.CASecret, namespace, nil, nil, nil, certBytes, privKeyBytes, istioCASecretType)
	// we need TypeMeta to create PropagationPolicy
	secret.TypeMeta = typemeta.Secret

	return secret, nil
}
