#!/bin/bash

# Copyright 2019 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# See the License for the specific language governing permissions and
# limitations under the License.

# Copyright 2014 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/../.. && pwd)
cd $ROOT

export GO111MODULE=off

TIMEOUT=${TIMEOUT:--timeout=600s}
GO_RACE=${GO_RACE:-}   # use GO_RACE="-race" to enable race testing
readonly GO_PACKAGE=github.com/pingcap/advanced-statefulset

test::find_dirs() {
  (
    cd "${ROOT}"
    find -L . -not \( \
        \( \
          -path './output/*' \
          -o -path './test/e2e/*' \
          -o -path './test/integration/*' \
          -o -path './vendor/*' \
        \) -prune \
      \) -name '*_test.go' -print0 | xargs -0n1 dirname | sed "s|^\./|${GO_PACKAGE}/|" | LC_ALL=C sort -u
  )
}

# Use eval to preserve embedded quoted strings.
testargs=()
eval "testargs=(${KUBE_TEST_ARGS:-})"

# Filter out arguments that start with "-" and move them to goflags.
testcases=()
for arg; do
  if [[ "${arg}" == -* ]]; then
    goflags+=("${arg}")
  else 
    testcases+=("${arg}")
  fi
done
if [[ ${#testcases[@]} -eq 0 ]]; then
  while IFS='' read -r line; do testcases+=("$line"); done < <(test::find_dirs)
fi 
set -- "${testcases[@]+${testcases[@]}}"

if [[ -n "${GO_RACE}" ]] ; then
  goflags+=("${GO_RACE}")
fi

runTests() {
    go test "${goflags[@]:+${goflags[@]}}" \
     "${TIMEOUT}" "${@}" \
     "${testargs[@]:+${testargs[@]}}" \
      && rc=$? || rc=$?
	return ${rc}
}

runTests "$@"
