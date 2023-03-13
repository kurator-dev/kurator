#!/bin/bash

# shellcheck disable=SC2164

HUGO_BASE_URL=${HUGO_BASE_URL:-"https://kurator.dev"}
export HUGO_ENV="production"
export HUGO_ENVIRONMENT="production"
export HUGO_KURATOR_VERSION=${KURATOR_VERSION:-"dev"}

pushd docs
    npm install --no-save
    hugo --minify --baseURL "${HUGO_BASE_URL}" --destination ../out/public
popd
