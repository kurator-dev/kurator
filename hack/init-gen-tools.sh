#!/usr/bin/env bash

# shellcheck disable=SC1090
set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)

JSONNET_BUNDLER_VERSION=${JSONNET_BUNDLER_VERSION:-v0.5.1}
JSONTOYAML_VERSION=${JSONTOYAML_VERSION:-v0.1.0}
JSONNET_VERSION=${JSONNET_VERSION:-v0.18.0}

source "${REPO_ROOT}/hack/util.sh"

# install prerequisite tools
if ! [ -x "$(command -v jb)" ]; then
    util::install_tools github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb "${JSONNET_BUNDLER_VERSION}"
fi
if ! [ -x "$(command -v gojsontoyaml)" ]; then
    util::install_tools github.com/brancz/gojsontoyaml "${JSONTOYAML_VERSION}"
fi
if ! [ -x "$(command -v jsonnet)" ]; then
    util::install_tools github.com/google/go-jsonnet/cmd/jsonnet "${JSONNET_VERSION}"
fi
