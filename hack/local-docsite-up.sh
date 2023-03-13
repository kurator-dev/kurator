#!/bin/bash

# shellcheck disable=SC2164

pushd docs
    export HUGO_KURATOR_VERSION=${KURATOR_VERSION:-"dev"}
    hugo serve
popd
