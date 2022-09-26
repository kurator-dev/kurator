#!/usr/bin/env bash

# shellcheck disable=SC1090

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
THANOS_OUT_PATH="${REPO_ROOT}/out/thanos"
THANOS_MANIFESTS_PATH="${REPO_ROOT}/manifests/profiles/thanos"
KUBE_THANOS_VER=${KUBE_THANOS_VER:-v0.26.0}

rm -rf "${THANOS_OUT_PATH}"
rm -rf "${THANOS_MANIFESTS_PATH}"
mkdir -p "${THANOS_MANIFESTS_PATH}"
mkdir -p "${THANOS_OUT_PATH}"
cp "${REPO_ROOT}/manifests/jsonnet/thanos/thanos.jsonnet" "${THANOS_OUT_PATH}/thanos.jsonnet"

echo 'begin to generate prom manifests'
echo "path: ${THANOS_OUT_PATH}";
echo "version: ${KUBE_THANOS_VER}"

pushd "${THANOS_OUT_PATH}"
    jb init
    jb install github.com/thanos-io/kube-thanos/jsonnet/kube-thanos@"${KUBE_THANOS_VER}"
    jb update

    cp "${REPO_ROOT}/hack/build-thanos.sh" build.sh

    bash build.sh thanos.jsonnet
popd

cp -r "${THANOS_OUT_PATH}"/manifests/* "${THANOS_MANIFESTS_PATH}"
