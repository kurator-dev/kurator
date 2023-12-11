#!/usr/bin/env bash

# shellcheck disable=SC1090

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
JB="${REPO_ROOT}/.tools/jb"
PROM_OUT_PATH=${REPO_ROOT}/out/prom
PROM_JSONNET_FILE=${REPO_ROOT}/$1
PROM_MANIFESTS_PATH=${REPO_ROOT}/${2}
KUBE_PROM_VER=${KUBE_PROM_VER:-v0.10.0}

echo 'begin to generate prom manifests'
echo "jsonnet: ${PROM_JSONNET_FILE}"
echo "manifetsts: ${PROM_MANIFESTS_PATH}";
echo "version: ${KUBE_PROM_VER}"

rm -rf "${PROM_OUT_PATH}"
rm -rf "${PROM_MANIFESTS_PATH}"
mkdir -p "${PROM_MANIFESTS_PATH}"
mkdir -p "${PROM_OUT_PATH}"
cp "${PROM_JSONNET_FILE}" "${PROM_OUT_PATH}/kube-prometheus.jsonnet"

pushd "${PROM_OUT_PATH}"
    ${JB} init
    ${JB} install github.com/prometheus-operator/kube-prometheus/jsonnet/kube-prometheus@"${KUBE_PROM_VER}"
    wget https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/"${KUBE_PROM_VER}"/build.sh -O build.sh
    ${JB} update

    PATH="${REPO_ROOT}/.tools:$PATH" bash build.sh kube-prometheus.jsonnet
popd

cp -r "${PROM_OUT_PATH}"/manifests/* "${PROM_MANIFESTS_PATH}"
