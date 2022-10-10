#!/usr/bin/env bash

# shellcheck disable=SC2046
set -o errexit
set -o nounset
set -o pipefail

kind delete clusters $(kind get clusters | grep kurator)
