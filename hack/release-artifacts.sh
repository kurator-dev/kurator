#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

GOOS=${GOOS:-"linux"}
GOARCH=${GOARCH:-"amd64"}

VERSION=${VERSION:-"latest"}
REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")
OUT_BASE_PATH=${OUT_BASE_PATH:-"${REPO_ROOT}/out"}
RELEASE_PATH="${OUT_BASE_PATH}/release-artifacts"
CHART_PATH="${OUT_BASE_PATH}/charts"

rm -rf "${RELEASE_PATH}"
mkdir -p "${RELEASE_PATH}"

BINS=("kurator")

# tar kurator binary
for BIN in "${BINS[@]}"; do
    echo "TAR BINARY: ${BIN}"
    BIN_RELEASE="${BIN}_${VERSION}_${GOOS}-${GOARCH}.tar.gz"
    pushd "${OUT_BASE_PATH}/${GOOS}-${GOARCH}/"
        tar -zcvf "${BIN_RELEASE}" "./${BIN}"
    popd
    mv "${OUT_BASE_PATH}/${GOOS}-${GOARCH}/${BIN_RELEASE}" "${RELEASE_PATH}"
done

# copy charts
cp -r "${CHART_PATH}"/*.tgz "${RELEASE_PATH}"
