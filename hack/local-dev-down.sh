#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

KURATOR_CLUSTERS=`kind get clusters | grep kurator`

kind delete clusters ${KURATOR_CLUSTERS}
