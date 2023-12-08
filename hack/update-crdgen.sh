#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
TOOL_DIR="${REPO_ROOT}/.tools"

CONTROLLER_GEN=${CONTROLLER_GEN:-"${TOOL_DIR}/controller-gen"}
KUSTOMIZE=${KUSTOMIZE:-"${TOOL_DIR}/kustomize"}
CRD_PATH=${CRD_PATH:-"manifests/config/crds"}

echo "Generating crd for cluster operator"
APIS_PATHS=("./pkg/apis/cluster/..." "./pkg/apis/infra/...")
OPERATOR_CHART_PATH=${OPERATOR_CHART_PATH:-"manifests/charts/cluster-operator"}

for APIS_PATH in "${APIS_PATHS[@]}"
do
    echo "Generating CRD for ${APIS_PATH}"
    ${CONTROLLER_GEN} crd paths="${APIS_PATH}" output:crd:dir="${CRD_PATH}"
done

echo "running kustomize to generate the final CRDs"
${KUSTOMIZE} "${CRD_PATH}" -o "${CRD_PATH}"/infrastructure.cluster.x-k8s.io_customclusters.yaml
mv "${CRD_PATH}"/*.yaml "${OPERATOR_CHART_PATH}"/crds/

echo "Generating crd for fleet manager"
APIS_PATHS=(
    "./pkg/apis/fleet/..."
    "./pkg/apis/apps/..."
    "./pkg/apis/backups/..."
    "./pkg/apis/pipeline/..."
)
FLEET_CHART_PATH=${FLEET_CHART_PATH:-"manifests/charts/fleet-manager"}
for APIS_PATH in "${APIS_PATHS[@]}"
do
    echo "Generating CRD for ${APIS_PATH}"
    ${CONTROLLER_GEN} crd:allowDangerousTypes=true paths="${APIS_PATH}" output:crd:dir="${CRD_PATH}"
done

mv "${CRD_PATH}"/*.yaml "${FLEET_CHART_PATH}"/crds/

