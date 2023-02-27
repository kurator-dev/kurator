#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

CONTROLLER_GEN=${CONTROLLER_GEN:-"$(go env GOPATH)/bin/controller-gen"}
CRD_PATH=${CRD_PATH:-"manifests/config/crds"}
WEBHOOK_PATH=${WEBHOOK_PATH:-"manifests/config/webhooks"}
APIS_PATHS=("./pkg/apis/cluster/..." "./pkg/apis/infra/...")
OPERATOR_CHART_PATH=${OPERATOR_CHART_PATH:-"manifests/charts/cluster-operator"}

for APIS_PATH in "${APIS_PATHS[@]}"
do
    echo "Generating CRD for ${APIS_PATH}"
    ${CONTROLLER_GEN} crd paths="${APIS_PATH}" output:crd:dir="${CRD_PATH}" \
        webhook output:webhook:dir="${WEBHOOK_PATH}"
done

echo "running kustomize to generate the final CRD"
kubectl kustomize "${CRD_PATH}" -o "${CRD_PATH}"/infrastructure.cluster.x-k8s.io_customclusters.yaml
mv "${CRD_PATH}"/*.yaml "manifests/charts/base/templates/"

echo "running kustomize to generate the final Webhook"
kubectl kustomize "${WEBHOOK_PATH}" -o "${WEBHOOK_PATH}"/manifests.yaml
mv "${WEBHOOK_PATH}"/manifests.yaml "${OPERATOR_CHART_PATH}"/templates/webhooks.yaml
