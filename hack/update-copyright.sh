#!/bin/bash

# Copyright 2022-2025 Kurator Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

WD=$(dirname "$0")
WD=$(cd "$WD"; pwd)

OLD_COPYRIGHT="Copyright Kurator Authors."
NEW_COPYRIGHT="Copyright 2022-2025 Kurator Authors."

echo "Searching for files with old copyright notice..."

find . -type f \( -name "*.go" -o -name "*.sh" -o -name "*.yaml" -o -name "*.yml" -o -name "*.txt" -o -name "*.md" \) -print0 | while IFS= read -r -d '' file; do
  if grep -q "$OLD_COPYRIGHT" "$file"; then
    echo "Updating copyright in: $file"
    sed -i "s|$OLD_COPYRIGHT|$NEW_COPYRIGHT|g" "$file"
  fi
done

echo "Copyright update completed."
