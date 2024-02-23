#!/usr/bin/env bash
# shellcheck disable=SC2086,SC1090,SC2206,SC1091
set -o errexit
set -o nounset
set -o pipefail

# This script installs Kurator cluster-operater and fleet-manager.

KUBECONFIG_PATH=${KUBECONFIG_PATH:-"${HOME}/.kube"}
MAIN_KUBECONFIG=${MAIN_KUBECONFIG:-"${KUBECONFIG_PATH}/kurator-host.config"}
export KUBECONFIG=${MAIN_KUBECONFIG}
COMMIT_ID=$(git rev-parse --short HEAD)
VERSION=$(echo "$COMMIT_ID" | grep -o '^[0-9]')

sleep 5s

helm repo add jetstack https://charts.jetstack.io
helm repo update
kubectl create namespace cert-manager
helm install -n cert-manager cert-manager jetstack/cert-manager --set installCRDs=true

helm repo add fluxcd-community https://fluxcd-community.github.io/helm-charts
cat <<EOF | helm install fluxcd fluxcd-community/flux2 --version 2.7.0 -n fluxcd-system --create-namespace -f -
imageAutomationController:
  create: false
imageReflectionController:
  create: false
notificationController:
  create: false
EOF

sleep 5s

VERSION=${VERSION} make docker
kind load docker-image ghcr.io/kurator-dev/cluster-operator:${VERSION} --name kurator-host
kind load docker-image ghcr.io/kurator-dev/fleet-manager:${VERSION} --name kurator-host

VERSION=${VERSION} make gen-chart
cd out/charts
helm install --create-namespace kurator-cluster-operator cluster-operator-${VERSION}.tgz -n kurator-system
helm install --create-namespace kurator-fleet-manager fleet-manager-${VERSION}.tgz -n kurator-system

echo "install kurator successful"
