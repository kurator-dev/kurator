#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

CONTROLLER_GEN=${CONTROLLER_GEN:-"$(go env GOPATH)/bin/controller-gen"}
CRD_PATH=${CRD_PATH:-"manifests/charts/base/templates"}
APIS_PATHS=("./pkg/apis/cluster/..." "./pkg/apis/infra/...")

for APIS_PATH in "${APIS_PATHS[@]}"
do
    echo "Generating CRD for ${APIS_PATH}"
    ${CONTROLLER_GEN} crd  paths="${APIS_PATH}" output:crd:dir="${CRD_PATH}"
done

kubectl kustomize manifests/charts/ -o "${CRD_PATH}"/cluster.kurator.dev_customclusters.yaml