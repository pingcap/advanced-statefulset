#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

export GO111MODULE=off

hack::ensure_ginkgo

GINKGO_PARALLEL=${GINKGO_PARALLEL:-n} # set to 'y' to run tests in parallel
# If 'y', Ginkgo's reporter will not print out in color when tests are run
# in parallel
GINKGO_NO_COLOR=${GINKGO_NO_COLOR:-n}

ginkgo_args=()

if [[ -n "${GINKGO_NODES:-}" ]]; then
    ginkgo_args+=("--nodes=${GINKGO_NODES}")
elif [[ ${GINKGO_PARALLEL} =~ ^[yY]$ ]]; then
    ginkgo_args+=("-p")
fi

if [[ "${GINKGO_NO_COLOR}" == "y" ]]; then
    ginkgo_args+=("--noColor")
fi

# We must precompile our e2e test, then it will be recognized as ginkgo test
# binary. Otherwise it will not be run parallelly.
# https://github.com/onsi/ginkgo/blob/v1.8.0/ginkgo/testsuite/test_suite.go#L101-L115
# https://github.com/onsi/ginkgo/blob/v1.8.0/ginkgo/testrunner/test_runner.go#L231-L245
go test -c -o output/bin/e2e.test ./test/e2e

echo "Running e2e tests:" >&2
$GINKGO_BIN "${ginkgo_args[@]:-}" output/bin/e2e.test -- \
    "${@:-}"
