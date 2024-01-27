#!/bin/bash

IMAGE_HUB=${IMAGE_HUB:-"ghcr.io/kurator-dev"}
IMAGE_TAG=${VERSION:-"$(VERSION)"}
SIGN_IMAGE=${SIGN_IMAGE:-"0"}

CLUSTER_OPERATOR_IMAGE=${CLUSTER_OPERATOR_IMAGE:-"${IMAGE_HUB}/cluster-operator:${IMAGE_TAG}"}
FLEET_MANAGER_IMAGE=${FLEET_MANAGER_IMAGE:-"${IMAGE_HUB}/fleet-manager:${IMAGE_TAG}"}

if [ $SIGN_IMAGE = "1" ]; then
    echo "Sign image: "${CLUSTER_OPERATOR_IMAGE}
    cosign sign --yes ${CLUSTER_OPERATOR_IMAGE}
    echo "Sign image: "${FLEET_MANAGER_IMAGE}
    cosign sign --yes ${FLEET_MANAGER_IMAGE}
else
    echo "Warning: The build image is not signed"
fi
    