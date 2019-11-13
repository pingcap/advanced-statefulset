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

source $ROOT/hack/lib.sh

readonly GO_PACKAGE=github.com/pingcap/advanced-statefulset

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
