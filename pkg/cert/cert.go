package cert

import v1 "k8s.io/api/core/v1"

type Generator interface {
	Secret(namespace string) (*v1.Secret, error)
}
