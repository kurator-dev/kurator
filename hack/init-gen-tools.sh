#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

source "${REPO_ROOT}/hack/util.sh"

# install prerequisite tools
if ! [ -x "$(command -v jb)" ]; then
    util::install_tools github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb latest
fi
if ! [ -x "$(command -v gojsontoyaml)" ]; then
    util::install_tools github.com/brancz/gojsontoyaml latest
fi
if ! [ -x "$(command -v jsonnet)" ]; then
    util::install_tools github.com/google/go-jsonnet/cmd/jsonnet latest
fi
