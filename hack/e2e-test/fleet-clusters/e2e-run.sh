#!/usr/bin/env bash

set -o errexit

REPO_ROOT=$(git rev-parse --show-toplevel)
DIR="$(cd "$(dirname "$0")" && pwd)"

"$DIR"/attachedcluster.sh