#!/usr/bin/env bash

# shellcheck disable=SC2086

set -o errexit
set -o nounset
set -o pipefail

PKG_PATH=kurator.dev/kurator/pkg/client-go
APIS_PATHS=(kurator.dev/kurator/pkg/apis/cluster/v1alpha1 kurator.dev/kurator/pkg/apis/infra/v1alpha1 kurator.dev/kurator/pkg/apis/fleet/v1alpha1 kurator.dev/kurator/pkg/apis/apps/v1alpha1)
ALL_APIS=$(IFS=, ; echo "${APIS_PATHS[*]}")

# For all commands, the working directory is the parent directory(repo root).
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

GOPATH=$(go env GOPATH | awk -F ':' '{print $1}')
export GOPATH
PATH=$PATH:$GOPATH/bin
export PATH
for APIS_PATH in "${APIS_PATHS[@]}"
do
  echo "Generating with deepcopy-gen for ${APIS_PATH}"
  deepcopy-gen \
    --go-header-file hack/boilerplate.go.txt \
    --input-dirs=${APIS_PATH} \
    --output-package=${APIS_PATH} \
    --output-file-base=zz_generated.deepcopy

  echo "Generating with register-gen for ${APIS_PATH}"
  register-gen \
    --go-header-file hack/boilerplate.go.txt \
    --input-dirs=${APIS_PATH} \
    --output-package=${APIS_PATH} \
    --output-file-base=zz_generated.register
done

echo "Generating with client-gen"
client-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-base="" \
  --input=${ALL_APIS} \
  --output-package=${PKG_PATH}/generated/clientset \
  --clientset-name=versioned

echo "Generating with lister-gen"
lister-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=${ALL_APIS} \
  --output-package=${PKG_PATH}/generated/listers

echo "Generating with informer-gen"
informer-gen \
  --go-header-file hack/boilerplate.go.txt \
  --input-dirs=${ALL_APIS} \
  --versioned-clientset-package=${PKG_PATH}/generated/clientset/versioned \
  --listers-package=${PKG_PATH}/generated/listers \
  --output-package=${PKG_PATH}/generated/informers
