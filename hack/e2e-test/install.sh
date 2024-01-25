#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# This script installs Kurator cluster-operater and fleet-manager.

KUBECONFIG_PATH=${KUBECONFIG_PATH:-"${HOME}/.kube"}
MAIN_KUBECONFIG=${MAIN_KUBECONFIG:-"${KUBECONFIG_PATH}/kurator-host.config"}
export KUBECONFIG=${MAIN_KUBECONFIG}
VERSION=${VERSION:-"0.6.0"}

helm repo add jetstack https://charts.jetstack.io
helm repo update
kubectl create namespace cert-manager
sleep 5s
helm install -n cert-manager cert-manager jetstack/cert-manager --set installCRDs=true

sleep 5s

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

helm repo add kurator https://kurator-dev.github.io/helm-charts
helm repo update
sleep 5s

helm install --create-namespace  kurator-cluster-operator kurator/cluster-operator --version=${VERSION} -n kurator-system 
helm install --create-namespace  kurator-fleet-manager kurator/fleet-manager --version=${VERSION} -n kurator-system 

echo "install kurator successful"
