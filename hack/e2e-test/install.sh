#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# This script installs Kurator cluster-operater and fleet-manager.

KUBECONFIG_PATH=${KUBECONFIG_PATH:-"${HOME}/.kube"}
MAIN_KUBECONFIG=${MAIN_KUBECONFIG:-"${KUBECONFIG_PATH}/kurator-host.config"}
export KUBECONFIG=${MAIN_KUBECONFIG}
VERSION=${VERSION:-"0.95.27"}

sleep 5s

VERSION=${VERSION} make docker
kind load docker-image ghcr.io/kurator-dev/cluster-operator:${VERSION} --name kurator-host
kind load docker-image ghcr.io/kurator-dev/fleet-manager:${VERSION} --name kurator-host

VERSION=${VERSION} make gen-chart
cd out/charts
helm install --create-namespace kurator-cluster-operator cluster-operator-${VERSION}.tgz -n kurator-system
helm install --create-namespace kurator-fleet-manager fleet-manager-${VERSION}.tgz -n kurator-system

echo "install kurator successful"
