#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

KURATOR_CLUSTERS=`docker ps -a --format '{{.Names}}' | grep kurator`

docker rm -f ${KURATOR_CLUSTERS}