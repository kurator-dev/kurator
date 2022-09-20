#!/usr/bin/env bash

# This script uses arg $1 (name of *.jsonnet file to use) to generate the manifests/*.yaml files.

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

echo 'begin to generate prom manifests'
echo "path: $1";
echo "version: $2"
echo "jsonnet file: $3"

pushd $1
    jb init
    jb install github.com/thanos-io/kube-thanos/jsonnet/kube-thanos@$2
    jb update

    cp ${REPO_ROOT}/hack/build-thanos.sh build.sh

    bash build.sh $3
popd