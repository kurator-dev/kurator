#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


# For all commands, the working directory is the parent directory(repo root).
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"
source hack/util.sh

CONFIG_PATH="${REPO_ROOT}/hack/crd-ref-docs/config.yaml"
OUT_DIR="${REPO_ROOT}/docs/content/en/references"
API_DIR="${REPO_ROOT}/pkg/apis"

if ! [ -x "$(command -v crd-ref-docs)" ]; then
    util::install_tools github.com/elastic/crd-ref-docs v0.0.8
fi

API_GROUPS=("cluster" "infra")

for APIGROUP in "${API_GROUPS[@]}"
do
    echo "Generating docs for ${APIGROUP}"
    crd-ref-docs \
    --config="${CONFIG_PATH}" \
    --source-path="${API_DIR}/${APIGROUP}" \
    --output-path="${OUT_DIR}/${APIGROUP}_types.md" \
    --max-depth 10 \
    --renderer=markdown
done