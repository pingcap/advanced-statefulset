#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/../.. && pwd)
cd $ROOT

export GO111MODULE=off

TIMEOUT=${TIMEOUT:--timeout=600s}
GO_RACE=${GO_RACE:-}   # use GO_RACE="-race" to enable race testing
readonly GO_PACKAGE=github.com/cofyc/advanced-statefulset

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
