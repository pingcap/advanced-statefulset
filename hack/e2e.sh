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
KUBE_VERSION=${KUBE_VERSION:-v1.16.3}
CLUSTER=${CLUSTER:-advanced-statefulset}

# https://github.com/kubernetes-sigs/kind/releases/tag/v0.6.0
declare -A kind_node_images
kind_node_images["v1.16.3"]="kindest/node:v1.16.3@sha256:bced4bc71380b59873ea3917afe9fb35b00e174d22f50c7cab9188eac2b0fb88"
kind_node_images["v1.12.10"]="kindest/node:v1.12.10@sha256:e93e70143f22856bd652f03da880bfc70902b736750f0a68e5e66d70d"

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
    image=""
    for v in ${!kind_node_images[*]}; do
        if [[ "$KUBE_VERSION" == "$v" ]]; then
            image=${kind_node_images[$v]}
            echo "info: image for $KUBE_VERSION: $image"
            break
        fi
    done
    if [ -z "$image" ]; then
        echo "error: no image for $KUBE_VERSION, exit"
        exit 1
    fi
    $KIND_BIN create cluster --name $CLUSTER --image kindest/node:$KUBE_VERSION --config hack/kindconfig.$KUBE_VERSION.yaml --loglevel debug
fi

export KUBECONFIG="$($KIND_BIN get kubeconfig-path --name="$CLUSTER")"
$KUBECTL_BIN cluster-info

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
