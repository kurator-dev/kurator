#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

CONTROLLER_GEN=${CONTROLLER_GEN:-"$(go env GOPATH)/bin/controller-gen"}
CRD_PATH=${CRD_PATH:-"manifests/charts/base/templates"}

${CONTROLLER_GEN} crd  paths="./pkg/apis/cluster/..." output:crd:dir="${CRD_PATH}"
