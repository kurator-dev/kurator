#!/usr/bin/env bash

# shellcheck disable=SC2086,SC1090,SC1091
set -o errexit
set -o nounset
set -o pipefail

ROOT_DIR=$(git rev-parse --show-toplevel)/hack
KIND_CONFIGS_ROOT=${ROOT_DIR}/kind-configs
source "${ROOT_DIR}"/util.sh

KIND_VERSION=${KIND_VERSION:-"kindest/node:v1.25.3"}

KUBECONFIG_PATH=${KUBECONFIG_PATH:-"${HOME}/.kube"}
MAIN_KUBECONFIG=${MAIN_KUBECONFIG:-"${KUBECONFIG_PATH}/config"}
CLUSTER_NAME=${HOST_CLUSTER_NAME:-"kurator"}
ENABLE_KIND_WITH_WORKER=${ENABLE_KIND_WITH_WORKER:-"false"}

TEMP_PATH=$(mktemp -d)
echo -e "Preparing kind config in path: ${TEMP_PATH}"
#When the Enable worker option is turned on, select to copy the configuration that contains the worker.
if [ ${ENABLE_KIND_WITH_WORKER} = "true" ]; then
    cp -rf ${ROOT_DIR}/kind-configs-with-worker/*.yaml "${TEMP_PATH}"/
else
    cp -rf "${KIND_CONFIGS_ROOT}"/*.yaml "${TEMP_PATH}"/
fi

util::create_cluster "${CLUSTER_NAME}" "${MAIN_KUBECONFIG}" "${KIND_VERSION}" "${TEMP_PATH}" "${TEMP_PATH}"/host.yaml

util::check_clusters_ready "${MAIN_KUBECONFIG}" "${CLUSTER_NAME}"

function print_success() {
  echo "Local clusters is running."
}

print_success
