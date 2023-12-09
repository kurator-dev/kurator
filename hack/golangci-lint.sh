#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
GOLANGCI_LINT_PKG="github.com/golangci/golangci-lint/cmd/golangci-lint"
GOLANGCI_LINT_VER="v1.51.2"

cd "${REPO_ROOT}"
source "hack/util.sh"

if ! [ -x "$(command -v golangci-lint)" ]; then
  util::install_tools ${GOLANGCI_LINT_PKG} ${GOLANGCI_LINT_VER}
fi

if golangci-lint run -c "$REPO_ROOT/common/config/.golangci.yml"; then
  echo 'Congratulations!  All Go source files have passed staticcheck.'
else
  echo # print one empty line, separate from warning messages.
  echo 'Please review the above warnings.'
  echo 'If the above warnings do not make sense, feel free to file an issue.'
  exit 1
fi
