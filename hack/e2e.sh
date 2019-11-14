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

source "${ROOT}/hack/lib.sh"

KEEP_CLUSTER=${KEEP_CLUSTER:-}
SKIP_BUILD=${SKIP_BUILD:-}
REUSE_CLUSTER=${REUSE_CLUSTER:-}
KUBE_VERSION=${KUBE_VERSION:-v1.16.1}
CLUSTER=${CLUSTER:-advanced-statefulset}

echo "KEEP_CLUSTER: $KEEP_CLUSTER"
echo "SKIP_BUILD: $SKIP_BUILD"
echo "REUSE_CLUSTER: $REUSE_CLUSTER"
echo "KUBE_VERSION: $KUBE_VERSION"
echo "CLUSTER: $CLUSTER"

hack::ensure_kind
hack::ensure_kubectl

trap 'cleanup' EXIT

function cleanup() {
    if [ -z "$KEEP_CLUSTER" ]; then
        if $KIND_BIN get clusters | grep $CLUSTER &>/dev/null; then
            echo "info: deleting the cluster '$CLUSTER'"
            $KIND_BIN delete cluster --name $CLUSTER
        fi
    fi
}

if $KIND_BIN get clusters | grep $CLUSTER &>/dev/null; then
    if [ -z "$REUSE_CLUSTER" ]; then
        echo "info: deleting the cluster '$CLUSTER'"
        $KIND_BIN delete cluster --name $CLUSTER
    else
        echo "info: reusing existing cluster"
    fi
fi

if ! $KIND_BIN get clusters | grep $CLUSTER &>/dev/null; then
    echo "info: creating the cluster '$CLUSTER'"
    $KIND_BIN create cluster --name $CLUSTER --image kindest/node:$KUBE_VERSION --config hack/kindconfig.$KUBE_VERSION.yaml
fi

export KUBECONFIG="$($KIND_BIN get kubeconfig-path --name="$CLUSTER")"
$KUBECTL_BIN cluster-info

if [ "$KUBE_VERSION" == "v1.12.10" ]; then
	# hack for https://github.com/coredns/coredns/issues/2391
	$KUBECTL_BIN -n kube-system set image deployment/coredns coredns=k8s.gcr.io/coredns:1.3.0
fi

export CONTROLLER_IMAGE=pingcap/advanced-statefulset:latest

if [ -z "$SKIP_BUILD" ]; then
	echo "info: building image $CONTROLLER_IMAGE"
	docker build -t $CONTROLLER_IMAGE .
else
	echo "info: skip building images"
fi

echo "info: loading image $CONTROLLER_IMAGE"
$KIND_BIN load docker-image --name $CLUSTER $CONTROLLER_IMAGE

hack/run-e2e.sh --kubectl-path=$KUBECTL_BIN \
    --kubeconfig=$KUBECONFIG \
	--provider=skeleton \
	--clean-start=true \
	--repo-root=$ROOT \
    "$@"
