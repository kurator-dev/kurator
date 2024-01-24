#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

KUBECONFIG_PATH=${KUBECONFIG_PATH:-"${HOME}/.kube"}
MAIN_KUBECONFIG=${MAIN_KUBECONFIG:-"${KUBECONFIG_PATH}/kurator-host.config"}
export KUBECONFIG=${MAIN_KUBECONFIG}

kubectl create secret generic kurator-member1 --from-file=kurator-member1.config=${KUBECONFIG_PATH}/kurator-member1.config
kubectl create secret generic kurator-member2 --from-file=kurator-member2.config=${KUBECONFIG_PATH}/kurator-member2.config

cat <<EOF | kubectl apply -f -
apiVersion: cluster.kurator.dev/v1alpha1
kind: AttachedCluster
metadata:
  name: kurator-member1
  namespace: default
spec:
  kubeconfig:
    name: kurator-member1
    key: kurator-member1.config
EOF

ok=false
sleep 10
kubectl get attachedclusters.cluster.kurator.dev kurator-member1 -o yaml | grep 'ready: true' && ok=true || ok=false
if [ ${ok} = false ]; then
    echo "create attachedCluster resources failed"
    exit 1
fi

cat <<EOF | kubectl apply -f -
apiVersion: cluster.kurator.dev/v1alpha1
kind: AttachedCluster
metadata:
  name: kurator-member2
  namespace: default
spec:
  kubeconfig:
    name: kurator-member2
    key: kurator-member2.config
EOF

ok=false
sleep 10
kubectl get attachedclusters.cluster.kurator.dev kurator-member1 -o yaml | grep 'ready: true' && ok=true || ok=false
if [ ${ok} = false ]; then
    echo "create attachedCluster resources failed"
    exit 1
fi
