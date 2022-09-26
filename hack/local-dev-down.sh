#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

kind delete clusters "$(kind get clusters | grep kurator)"
