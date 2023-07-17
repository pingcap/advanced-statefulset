#!/usr/bin/env bash

# Copyright 2020 PingCAP, Inc.
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

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source hack/lib.sh
hack::ensure_codegen

function codegen::join() { local IFS="$1"; shift; echo "$*"; }

# `--output-base $ROOT` will output generated code to current dir
GOBIN=$OUTPUT_BIN bash $ROOT/hack/generate-groups.sh "deepcopy,defaulter,client,informer,lister,applyconfiguration" \
  github.com/pingcap/advanced-statefulset/client/client \
  github.com/pingcap/advanced-statefulset/client/apis \
  "apps:v1" \
  --output-base $ROOT \
  --go-header-file "${ROOT}"/../hack/boilerplate/boilerplate.k8s.go.txt

# cp zz_generated.deepcopy.go
cp github.com/pingcap/advanced-statefulset/client/apis/apps/v1/zz_generated.deepcopy.go $ROOT/apis/apps/v1/zz_generated.deepcopy.go

# cp zz_generated.defaults.go
cp github.com/pingcap/advanced-statefulset/client/apis/apps/v1/zz_generated.defaults.go $ROOT/apis/apps/v1/zz_generated.defaults.go

# then we merge generated code with our code base and clean up
cp -r github.com/pingcap/advanced-statefulset/client/client $ROOT && rm -rf github.com

# work around for https://github.com/kubernetes/code-generator/issues/84
git checkout client/listers/apps/v1/expansion_generated.go
