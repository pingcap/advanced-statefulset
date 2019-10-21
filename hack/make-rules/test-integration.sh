#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/../.. && pwd)
cd $ROOT

source $ROOT/hack/lib.sh

readonly GO_PACKAGE=github.com/cofyc/advanced-statefulset

# Give integration tests longer to run by default.
KUBE_TIMEOUT=${KUBE_TIMEOUT:--timeout=600s}
KUBE_TEST_ARGS=${KUBE_TEST_ARGS:-}
# Default glog module settings.
KUBE_TEST_VMODULE=${KUBE_TEST_VMODULE:-"statefulset*=6"}

test::find_integration_test_dirs() {
  (
    cd "${ROOT}"
    find test/integration/ -name '*_test.go' -print0 \
      | xargs -0n1 dirname | sed "s|^|${GO_PACKAGE}/|" \
      | LC_ALL=C sort -u
  )
}

hack::install_etcd

export PATH="$(pwd)/output/etcd:${PATH}"

# export GO_RACE
#
# Enable the Go race detector.
export GO_RACE="-race"
make -C "${ROOT}" test \
	WHAT="${WHAT:-$(test::find_integration_test_dirs | paste -sd' ' -)}" \
	GOFLAGS="${GOFLAGS:-}" \
	KUBE_TEST_ARGS="--alsologtostderr=true ${KUBE_TEST_ARGS:-} ${SHORT:--short=true} --vmodule=${KUBE_TEST_VMODULE}" \
	KUBE_TIMEOUT="${KUBE_TIMEOUT}"
