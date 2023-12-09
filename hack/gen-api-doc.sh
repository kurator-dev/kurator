#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


# For all commands, the working directory is the parent directory(repo root).
REPO_ROOT=$(git rev-parse --show-toplevel)
GEN_CRD_API_REFERENCE_DOCS=${REPO_ROOT}/.tools/gen-crd-api-reference-docs

CONFIG_FILE="${REPO_ROOT}/hack/api-docs/config.json"
TEMPLATE_DIR="${REPO_ROOT}/hack/api-docs/template"
OUT_DIR="${REPO_ROOT}/docs/content/en/references"
API_DIR="./pkg/apis"
API_GROUPS=("cluster" "infra" "fleet" "apps" "backups" "pipeline")

for APIGROUP in "${API_GROUPS[@]}"
do
    OUT_FILE="${OUT_DIR}/${APIGROUP}_v1alpha1_types.html"
    echo "Generating docs for ${APIGROUP}/v1alpha1 to ${OUT_FILE}"
    ${GEN_CRD_API_REFERENCE_DOCS} \
      --api-dir="${API_DIR}/${APIGROUP}/v1alpha1" \
      --config="${CONFIG_FILE}" \
      --template-dir="${TEMPLATE_DIR}" \
      --out-file="${OUT_FILE}"
done
