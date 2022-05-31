#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

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