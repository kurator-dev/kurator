#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
GOLANGCI_LINT=${REPO_ROOT}/.tools/golangci-lint

if ${GOLANGCI_LINT} run -c "$REPO_ROOT/common/config/.golangci.yml"; then
  echo 'Congratulations!  All Go source files have passed staticcheck.'
else
  echo # print one empty line, separate from warning messages.
  echo 'Please review the above warnings.'
  echo 'If the above warnings do not make sense, feel free to file an issue.'
  exit 1
fi
