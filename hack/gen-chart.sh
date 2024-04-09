#!/bin/bash

# shellcheck disable=SC2046,SC2086

set -e

REPO_ROOT=$(git rev-parse --show-toplevel)

OUT_BASE_PATH=${REPO_ROOT}/out
CHART_OUT_PATH=${OUT_BASE_PATH}/charts

rm -rf "${OUT_BASE_PATH}"/charts
mkdir -p "${OUT_BASE_PATH}"/charts

MAINIFESTS_CHART_PATH=${REPO_ROOT}/manifests/charts
HELM_CHARTS=(cluster-operator fleet-manager)
HELM_CHART_VERSION=${HELM_CHART_VERSION:-"0.1.0"}
IMAGE_HUB=${IMAGE_HUB:-"ghcr.io/kurator-dev"}
IMAGE_TAG=${IMAGE_TAG:-"latest"}

source "$REPO_ROOT/hack/util.sh"

for c in "${HELM_CHARTS[@]}"
do
    echo "gen chart $c"
    cp -r "${MAINIFESTS_CHART_PATH}/${c}" "${CHART_OUT_PATH}/${c}"
    util::sed_in_place "s|hub: ghcr.io/kurator-dev|hub: ${IMAGE_HUB}|g" "$(find ${CHART_OUT_PATH}/${HELM_CHART_NAME} -type f | grep values.yaml)"
    util::sed_in_place "s|tag: latest|tag: ${IMAGE_TAG}|g" "$(find ${CHART_OUT_PATH}/${HELM_CHART_NAME} -type f | grep values.yaml)"
    util::sed_in_place "s|version: 0.1.0|version: ${HELM_CHART_VERSION}|g" "$(find ${CHART_OUT_PATH}/${HELM_CHART_NAME} -type f | grep Chart.yaml)"
    util::sed_in_place "s|appVersion: 0.1.0|appVersion: ${HELM_CHART_VERSION}|g" "$(find ${CHART_OUT_PATH}/${HELM_CHART_NAME} -type f | grep Chart.yaml)"
    helm package "${CHART_OUT_PATH}/${c}" -d "${CHART_OUT_PATH}"
done



