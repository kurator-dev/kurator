#!/usr/bin/env bash

# shellcheck disable=SC2086,SC1090,SC2206,SC1091
set -o errexit
set -o nounset
set -o pipefail

# This script starts a local karmada control plane based on current codebase and with a certain number of clusters joined.
# Parameters: [HOST_IPADDRESS](optional) if you want to export clusters' API server port to specific IP address
# This script depends on utils in: ${REPO_ROOT}/hack/util.sh
# 1. used by developer to setup develop environment quickly.
# 2. used by e2e testing to setup test environment automatically.
ROOT_DIR=$(git rev-parse --show-toplevel)/hack
KIND_CONFIGS_ROOT=${ROOT_DIR}/kind-configs
source "${ROOT_DIR}"/util.sh

KIND_VERSION=${KIND_VERSION:-"kindest/node:v1.27.3"}

# variable define
KUBECONFIG_PATH=${KUBECONFIG_PATH:-"${HOME}/.kube"}
MAIN_KUBECONFIG=${MAIN_KUBECONFIG:-"${KUBECONFIG_PATH}/kurator-host.config"}
HOST_CLUSTER_NAME=${HOST_CLUSTER_NAME:-"kurator-host"}
MEMBER_CLUSTER_KUBECONFIG=${MEMBER_CLUSTER_KUBECONFIG:-"${KUBECONFIG_PATH}/kurator-member.config"}
MEMBER_CLUSTER_NAME=${MEMBER_CLUSTER_NAME:-"kurator-member"}
ENABLE_KIND_WITH_WORKER=${ENABLE_KIND_WITH_WORKER:-"false"}

#prepare for kind cluster config
TEMP_PATH=$(mktemp -d)
echo -e "Preparing kind config in path: ${TEMP_PATH}"
#When the Enable worker option is turned on, select to copy the configuration that contains the worker.
if [ ${ENABLE_KIND_WITH_WORKER} = "true" ]; then
    cp -rf ${ROOT_DIR}/kind-configs-with-worker/*.yaml "${TEMP_PATH}"/
else
    cp -rf "${KIND_CONFIGS_ROOT}"/*.yaml "${TEMP_PATH}"/
fi

util::create_cluster "${HOST_CLUSTER_NAME}" "${MAIN_KUBECONFIG}" "${KIND_VERSION}" "${TEMP_PATH}" "${TEMP_PATH}"/host.yaml
util::create_cluster "${MEMBER_CLUSTER_NAME}" "${MEMBER_CLUSTER_KUBECONFIG}" "${KIND_VERSION}" "${TEMP_PATH}" "${TEMP_PATH}"/member1.yaml

util::check_clusters_ready "${MAIN_KUBECONFIG}" "${HOST_CLUSTER_NAME}"
sleep 5s
util::check_clusters_ready "${MEMBER_CLUSTER_KUBECONFIG}" "${MEMBER_CLUSTER_NAME}"
sleep 10s

# connecting networks between primary, remote clusters
echo "connect primary <-> remote"
util::connect_kind_clusters "${HOST_CLUSTER_NAME}" "${MAIN_KUBECONFIG}" "${MEMBER_CLUSTER_NAME}" "${MEMBER_CLUSTER_KUBECONFIG}" 1

echo "cluster networks connected"

echo "install metallb in host cluster"
util::install_metallb ${MAIN_KUBECONFIG} ${HOST_CLUSTER_NAME} "ipv4" "255"


echo "starting install metallb in member clusters"
MEMBER_CLUSTERS=(${MEMBER_CLUSTER_NAME})
MEMBER_KUBECONFIGS=(${MEMBER_CLUSTER_KUBECONFIG})
MEMBER_IPSPACES=("254" "253")
echo "install metallb in ${MEMBER_CLUSTERS}"
util::install_metallb ${MEMBER_KUBECONFIGS} ${MEMBER_CLUSTERS} "ipv4" ${MEMBER_IPSPACES}

function print_success() {
  echo "Local clusters is running."
  echo -e "\nTo start using your host cluster, run:"
  echo -e "  export KUBECONFIG=${MAIN_KUBECONFIG}"
  echo -e "\nTo manage your remote clusters, run:"
  echo -e "  export KUBECONFIG=${MEMBER_CLUSTER_KUBECONFIG}"
}

print_success