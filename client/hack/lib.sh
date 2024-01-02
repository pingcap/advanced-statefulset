#!/usr/bin/env bash

# Copyright 2023 PingCAP, Inc.
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

if [ -z "$ROOT" ]; then
    echo "error: ROOT should be initialized"
    exit 1
fi

OUTPUT=${ROOT}/output
OUTPUT_BIN=${OUTPUT}/bin

K8S_VERSION=${K8S_VERSION:-0.26.12}

function hack::ensure_codegen() {
    echo "Installing codegen..."
    GOBIN=$OUTPUT_BIN go install k8s.io/code-generator/cmd/{defaulter-gen,client-gen,lister-gen,informer-gen,deepcopy-gen,applyconfiguration-gen}@v$K8S_VERSION
}
