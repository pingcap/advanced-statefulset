#!/usr/bin/env bash

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

# Copyright 2019 The Kubernetes Authors.
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

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

export GO111MODULE=off

function codegen::join() { local IFS="$1"; shift; echo "$*"; }

bash "${CODEGEN_PKG}"/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/pingcap/advanced-statefulset/pkg/client \
  github.com/pingcap/advanced-statefulset/pkg/apis \
  "apps:v1alpha1 apps:v1" \
  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate/boilerplate.k8s.go.txt

# work around for https://github.com/kubernetes/code-generator/issues/84
git checkout pkg/client/listers/apps/v1alpha1/expansion_generated.go
git checkout pkg/client/listers/apps/v1/expansion_generated.go

#
# This requires GOPATH/src/k8s.io/kubernetes/vendor/k8s.io/api/core/v1 to exist.
# We run it manually for now.
# TODO: fix it
#
# EXT_FQ_APIS=(
    # github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1alpha1
    # github.com/pingcap/advanced-statefulset/pkg/apis/apps/v1
    # github.com/pingcap/advanced-statefulset/vendor/k8s.io/kubernetes/pkg/apis/core/v1
# )

# "${GOPATH}/bin/defaulter-gen"  \
    # --input-dirs "$(codegen::join , "${EXT_FQ_APIS[@]}")" \
    # -O zz_generated.defaults  \
    # --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate/boilerplate.k8s.go.txt \
    # -v 5
