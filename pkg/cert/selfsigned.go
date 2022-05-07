package cert

import (
	"fmt"
	"time"

	k8ssecret "istio.io/istio/security/pkg/k8s/secret"
	"istio.io/istio/security/pkg/pki/ca"
	"istio.io/istio/security/pkg/pki/util"
	v1 "k8s.io/api/core/v1"
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

	secret := k8ssecret.BuildSecret("", ca.CASecret, namespace, nil, nil, nil, certBytes, privKeyBytes, istioCASecretType)

	return secret, nil
}
