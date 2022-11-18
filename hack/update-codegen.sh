#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

PKG_PATH=kurator.dev/kurator/pkg/client-go
APIS_PATH=kurator.dev/kurator/pkg/apis/cluster/v1alpha1

# For all commands, the working directory is the parent directory(repo root).
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

GOPATH=$(go env GOPATH | awk -F ':' '{print $1}')
export GOPATH
PATH=$PATH:$GOPATH/bin
export PATH

echo "Generating with deepcopy-gen"
deepcopy-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=${APIS_PATH} \
  --output-package=${APIS_PATH} \
  --output-file-base=zz_generated.deepcopy

echo "Generating with register-gen"
register-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=${APIS_PATH} \
  --output-package=${APIS_PATH} \
  --output-file-base=zz_generated.register

echo "Generating with client-gen"
client-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-base="" \
  --input=${APIS_PATH} \
  --output-package=${PKG_PATH}/generated/clientset \
  --clientset-name=versioned

echo "Generating with lister-gen"
lister-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=${APIS_PATH} \
  --output-package=${PKG_PATH}/generated/listers

echo "Generating with informer-gen"
informer-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=${APIS_PATH} \
  --versioned-clientset-package=${PKG_PATH}/generated/clientset/versioned \
  --listers-package=${PKG_PATH}/generated/listers \
  --output-package=${PKG_PATH}/generated/informers
