#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


# For all commands, the working directory is the parent directory(repo root).
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"
source hack/util.sh

CONFIG_FILE="${REPO_ROOT}/hack/api-docs/config.json"
TEMPLATE_DIR="${REPO_ROOT}/hack/api-docs/template"
OUT_DIR="${REPO_ROOT}/docs/content/en/references"
API_DIR="./pkg/apis"

if ! [ -x "$(command -v gen-crd-api-reference-docs)" ]; then
    util::install_tools github.com/ahmetb/gen-crd-api-reference-docs 45bac9a # 2023-03-28
fi

API_GROUPS=("cluster" "infra" "fleet")

for APIGROUP in "${API_GROUPS[@]}"
do
    echo "Generating docs for ${APIGROUP}/v1alpha1 to ${OUT_DIR}/${APIGROUP}_types.md"
    gen-crd-api-reference-docs \
      --api-dir="${API_DIR}/${APIGROUP}/v1alpha1" \
      --config="${CONFIG_FILE}" \
      --template-dir="${TEMPLATE_DIR}" \
      --out-file="${OUT_DIR}/${APIGROUP}_v1alpha1_types.html"
done
