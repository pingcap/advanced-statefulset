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

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source $ROOT/hack/lib.sh

export GO111MODULE=off

go install ./vendor/k8s.io/code-generator/cmd/openapi-gen

function codegen::join() { local IFS="$1"; shift; echo "$*"; }

pkgs=($(find vendor/k8s.io/api -mindepth 2 -maxdepth 2 | xargs -n 1 -Ipkg echo github.com/pingcap/advanced-statefulset/pkg))
pkgs+=(
    github.com/pingcap/advanced-statefulset/vendor/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1
    github.com/pingcap/advanced-statefulset/vendor/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1
	github.com/pingcap/advanced-statefulset/vendor/k8s.io/kube-aggregator/pkg/apis/apiregistration/v1
	github.com/pingcap/advanced-statefulset/vendor/k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1
)
pkgs+=(
    github.com/pingcap/advanced-statefulset/vendor/k8s.io/apimachinery/pkg/apis/meta/v1
    github.com/pingcap/advanced-statefulset/vendor/k8s.io/apimachinery/pkg/runtime
    github.com/pingcap/advanced-statefulset/vendor/k8s.io/apimachinery/pkg/version
    github.com/pingcap/advanced-statefulset/vendor/k8s.io/apimachinery/pkg/api/resource
    github.com/pingcap/advanced-statefulset/vendor/k8s.io/apimachinery/pkg/util/intstr
)

$GOPATH/bin/openapi-gen  \
    --v 1 \
    --logtostderr \
    --input-dirs "$(codegen::join , "${pkgs[@]}")" \
    --output-package "github.com/pingcap/advanced-statefulset/vendor/k8s.io/kubernetes/pkg/generated/openapi" \
    -O zz_generated.openapi \
    --go-header-file $ROOT/hack/boilerplate/boilerplate.k8s.go.txt \
    -r output/KUBE_violations.report
