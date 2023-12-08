#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


# For all commands, the working directory is the parent directory(repo root).
REPO_ROOT=$(git rev-parse --show-toplevel)
TOOLS_DIR="${REPO_ROOT}/.tools"

KUSTOMIZE_VERSION=v4.5.5
CONTROLLER_GEN_VERSION=v0.8.0
CODE_GENERATOR_VERSION=v0.25.2

GOBIN=${TOOLS_DIR} go install sigs.k8s.io/kustomize/kustomize/v4@${KUSTOMIZE_VERSION}
GOBIN=${TOOLS_DIR} go install sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION}
GOBIN=${TOOLS_DIR} go install k8s.io/code-generator/cmd/{deepcopy-gen,client-gen,lister-gen,informer-gen,register-gen}@${CODE_GENERATOR_VERSION}
