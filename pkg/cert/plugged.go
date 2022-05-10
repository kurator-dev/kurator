package cert

import (
	"os"
	"path"

	"istio.io/istio/security/pkg/pki/ca"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cacertsSecretName = "cacerts"
)

type PluggedCert struct {
	dir string
}

func NewPluggedCert(basePath string) *PluggedCert {
	return &PluggedCert{
		dir: basePath,
	}
}

func (cert *PluggedCert) Secret(namespace string) (*v1.Secret, error) {
	caCert, err := os.ReadFile(path.Join(cert.dir, ca.CACertFile))
	if err != nil {
		return nil, err
	}

	caKey, _ := os.ReadFile(path.Join(cert.dir, ca.CAPrivateKeyFile))
	if err != nil {
		return nil, err
	}

	rootCert, _ := os.ReadFile(path.Join(cert.dir, ca.RootCertFile))
	if err != nil {
		return nil, err
	}

	certChain, _ := os.ReadFile(path.Join(cert.dir, ca.CertChainFile))
	if err != nil {
		return nil, err
	}

	return &v1.Secret{
		// we need TypeMeta to create PropagationPolicy
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cacertsSecretName,
			Namespace: namespace,
		},
		StringData: map[string]string{
			ca.CACertFile:       string(caCert),
			ca.CAPrivateKeyFile: string(caKey),
			ca.RootCertFile:     string(rootCert),
			ca.CertChainFile:    string(certChain),
		},
	}, nil
}
