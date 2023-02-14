#!/bin/bash

# Copyright Istio Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# shellcheck disable=SC2046,SC2086

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

OUT_BASE_PATH=${REPO_ROOT}/out
CHART_OUT_PATH=${OUT_BASE_PATH}/charts

rm -rf "${OUT_BASE_PATH}"/charts
mkdir -p "${OUT_BASE_PATH}"/charts

MAINIFESTS_CHART_PATH=${REPO_ROOT}/manifests/charts
HELM_CHARTS=(base cluster-operator)
HELM_CHART_VERSION=${HELM_CHART_VERSION:-"0.1.0"}
IMAGE_HUB=${IMAGE_HUB:-"ghcr.io/kurator-dev"}
IMAGE_TAG=${IMAGE_TAG:-"latest"}

for c in "${HELM_CHARTS[@]}"
do
    echo "gen chart $c"
    cp -r "${MAINIFESTS_CHART_PATH}/${c}" "${CHART_OUT_PATH}/${c}"
    sed -i "s|hub: ghcr.io/kurator-dev|hub: ${IMAGE_HUB}|g" $(find ${CHART_OUT_PATH}/${HELM_CHART_NAME} -type f | grep values.yaml)
    sed -i "s|tag: latest|tag: ${IMAGE_TAG}|g" $(find ${CHART_OUT_PATH}/${HELM_CHART_NAME} -type f | grep values.yaml)
    sed -i "s|version: 0.1.0|version: ${HELM_CHART_VERSION}|g" $(find ${CHART_OUT_PATH}/${HELM_CHART_NAME} -type f | grep Chart.yaml)
    sed -i "s|appVersion: 0.1.0|appVersion: ${HELM_CHART_VERSION}|g" $(find ${CHART_OUT_PATH}/${HELM_CHART_NAME} -type f | grep Chart.yaml)
    helm package "${CHART_OUT_PATH}/${c}" -d "${CHART_OUT_PATH}"
done



