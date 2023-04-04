#!/bin/bash

set -e

ec=0
for fn in "$@"; do
  if ! grep -L -q -i "License" "${fn}"; then
    echo "Missing license: ${fn}"
    ec=1
  fi

  if ! grep -L -q -e "Copyright" "${fn}"; then
    echo "Missing copyright: ${fn}"
    ec=1
  fi
done

exit $ec
