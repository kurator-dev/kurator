#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

source "${REPO_ROOT}/hack/util.sh"

# install prerequisite tools
if ! [ -x "$(command -v jb)" ]; then
    util::install_tools github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb v0.5.1
fi
if ! [ -x "$(command -v gojsontoyaml)" ]; then
    util::install_tools github.com/brancz/gojsontoyaml v0.1.0
fi
if ! [ -x "$(command -v jsonnet)" ]; then
util::install_tools github.com/google/go-jsonnet/cmd/jsonnet v0.18.0
fi

echo 'begin to generate prom manifests'
echo "path: $1";
echo "version: $2"
echo "jsonnet file: $3"

pushd $1
    jb init
    jb install github.com/prometheus-operator/kube-prometheus/jsonnet/kube-prometheus@$2
    wget https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/$2/build.sh -O build.sh
    jb update

    bash build.sh $3
popd