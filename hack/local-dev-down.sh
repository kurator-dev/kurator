#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

UBRAIN_CLUSTERS=`docker ps -a --format '{{.Names}}' | grep ubrain`

docker rm -f ${UBRAIN_CLUSTERS}