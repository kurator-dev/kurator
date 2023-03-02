#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

CONTROLLER_GEN=${CONTROLLER_GEN:-"$(go env GOPATH)/bin/controller-gen"}
CRD_PATH=${CRD_PATH:-"manifests/config/crds"}
APIS_PATHS=("./pkg/apis/cluster/..." "./pkg/apis/infra/...")
OPERATOR_CHART_PATH=${OPERATOR_CHART_PATH:-"manifests/charts/cluster-operator"}

for APIS_PATH in "${APIS_PATHS[@]}"
do
    echo "Generating CRD for ${APIS_PATH}"
    ${CONTROLLER_GEN} crd paths="${APIS_PATH}" output:crd:dir="${CRD_PATH}"
done

echo "running kustomize to generate the final CRDs"
kubectl kustomize "${CRD_PATH}" -o "${CRD_PATH}"/infrastructure.cluster.x-k8s.io_customclusters.yaml
mv "${CRD_PATH}"/*.yaml "${OPERATOR_CHART_PATH}"/crds/
