#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

# This script starts a local karmada control plane based on current codebase and with a certain number of clusters joined.
# Parameters: [HOST_IPADDRESS](optional) if you want to export clusters' API server port to specific IP address
# This script depends on utils in: ${REPO_ROOT}/hack/util.sh
# 1. used by developer to setup develop environment quickly.
# 2. used by e2e testing to setup test environment automatically.
REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")
KIND_CONFIGS_ROOT=${REPO_ROOT}/kind-configs
source "${REPO_ROOT}"/util.sh

METALLB_VERSION=${METALLB_VERSION:-"v0.10.2"}
KIND_VERSION=${KIND_VERSION:-"kindest/node:v1.23.4"}

# variable define
KUBECONFIG_PATH=${KUBECONFIG_PATH:-"${HOME}/.kube"}
MAIN_KUBECONFIG=${MAIN_KUBECONFIG:-"${KUBECONFIG_PATH}/kurator-host.config"}
MEMBER_CLUSTER_KUBECONFIG=${MEMBER_CLUSTER_KUBECONFIG:-"${KUBECONFIG_PATH}/kurator-members.config"}
HOST_CLUSTER_NAME=${HOST_CLUSTER_NAME:-"kurator-host"}
MEMBER_CLUSTER_1_NAME=${MEMBER_CLUSTER_1_NAME:-"kurator-member1"}
MEMBER_CLUSTER_2_NAME=${MEMBER_CLUSTER_2_NAME:-"kurator-member2"}
HOST_IPADDRESS=${1:-}

#prepare for kind cluster config
TEMP_PATH=$(mktemp -d)
echo -e "Preparing kind config in path: ${TEMP_PATH}"
cp -rf "${KIND_CONFIGS_ROOT}"/*.yaml "${TEMP_PATH}"/

#By default, the cluster created is a single-node cluster. If the command parameter “aw” or “add-worker” is added, the cluster to be created contains an additional worker node.
if [ $# -gt 0 ] && [ $1 = "aw" -o $1 = "add-worker" ]; then
    echo "Multi-node cluster will be created ."
    util::create_cluster "${HOST_CLUSTER_NAME}" "${MAIN_KUBECONFIG}" "${KIND_VERSION}" "${TEMP_PATH}" "${TEMP_PATH}"/host-add-worker.yaml
    util::create_cluster "${MEMBER_CLUSTER_1_NAME}" "${MEMBER_CLUSTER_KUBECONFIG}" "${KIND_VERSION}" "${TEMP_PATH}" "${TEMP_PATH}"/member1-add-worker.yaml
    util::create_cluster "${MEMBER_CLUSTER_2_NAME}" "${MEMBER_CLUSTER_KUBECONFIG}" "${KIND_VERSION}" "${TEMP_PATH}" "${TEMP_PATH}"/member2-add-worker.yaml
else
    echo "Single-node cluster will be created ."
    util::create_cluster "${HOST_CLUSTER_NAME}" "${MAIN_KUBECONFIG}" "${KIND_VERSION}" "${TEMP_PATH}" "${TEMP_PATH}"/host.yaml
    util::create_cluster "${MEMBER_CLUSTER_1_NAME}" "${MEMBER_CLUSTER_KUBECONFIG}" "${KIND_VERSION}" "${TEMP_PATH}" "${TEMP_PATH}"/member1.yaml
    util::create_cluster "${MEMBER_CLUSTER_2_NAME}" "${MEMBER_CLUSTER_KUBECONFIG}" "${KIND_VERSION}" "${TEMP_PATH}" "${TEMP_PATH}"/member2.yaml
fi


util::check_clusters_ready "${MAIN_KUBECONFIG}" "${HOST_CLUSTER_NAME}"
sleep 5s
util::check_clusters_ready "${MEMBER_CLUSTER_KUBECONFIG}" "${MEMBER_CLUSTER_1_NAME}"
sleep 5s
util::check_clusters_ready "${MEMBER_CLUSTER_KUBECONFIG}" "${MEMBER_CLUSTER_2_NAME}"
sleep 5s

# connecting networks between primary, remote1 and remote2 clusters
echo "connect remote1 <-> remote2"
util::connect_kind_clusters "${MEMBER_CLUSTER_1_NAME}" "${MEMBER_CLUSTER_KUBECONFIG}" "${MEMBER_CLUSTER_2_NAME}" "${MEMBER_CLUSTER_KUBECONFIG}" 1

echo "connect primary <-> remote1"
util::connect_kind_clusters "${HOST_CLUSTER_NAME}" "${MAIN_KUBECONFIG}" "${MEMBER_CLUSTER_1_NAME}" "${MEMBER_CLUSTER_KUBECONFIG}" 1

echo "connect primary <-> remote2"
util::connect_kind_clusters "${HOST_CLUSTER_NAME}" "${MAIN_KUBECONFIG}" "${MEMBER_CLUSTER_2_NAME}" "${MEMBER_CLUSTER_KUBECONFIG}" 1

echo "cluster networks connected"

echo "install metallb in host cluster"
kubectl create ns metallb-system --kubeconfig="${MAIN_KUBECONFIG}" --context="${HOST_CLUSTER_NAME}"
util::install_metallb ${MAIN_KUBECONFIG} ${HOST_CLUSTER_NAME}
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/${METALLB_VERSION}/manifests/metallb.yaml --kubeconfig="${MAIN_KUBECONFIG}" --context="${HOST_CLUSTER_NAME}"

echo "starting install metallb in member clusters"
MEMBER_CLUSTERS=(${MEMBER_CLUSTER_1_NAME} ${MEMBER_CLUSTER_2_NAME})
for c in ${MEMBER_CLUSTERS[@]}
do
  echo "install metallb in $c"
  kubectl create ns metallb-system --kubeconfig="${MEMBER_CLUSTER_KUBECONFIG}" --context="${c}"
  util::install_metallb ${MEMBER_CLUSTER_KUBECONFIG} ${c}
  kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/${METALLB_VERSION}/manifests/metallb.yaml --kubeconfig="${MEMBER_CLUSTER_KUBECONFIG}" --context="${c}"
done



function print_success() {
  echo "Local clusters is running."
  echo -e "\nTo start using your host cluster, run:"
  echo -e "  export KUBECONFIG=${MAIN_KUBECONFIG}"
  echo -e "\nTo manage your remote clusters, run:"
  echo -e "  export KUBECONFIG=${MEMBER_CLUSTER_KUBECONFIG}"
  echo "Please use 'kubectl config use-context member1/member2' to switch to the different remote cluster."
}

print_success