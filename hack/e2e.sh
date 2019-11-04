#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

KEEP_CLUSTER=${KEEP_CLUSTER:-}
SKIP_BUILD=${SKIP_BUILD:-}
REUSE_CLUSTER=${REUSE_CLUSTER:-}
KIND_IMAGE=${KIND_IMAGE:-kindest/node:v1.16.1}
CLUSTER=${CLUSTER:-advanced-statefulset}

if [ -z "$KIND_IMAGE" ]; then
    echo "error: KIND_IMAGE not specified"
    exit 1
fi

echo "KEEP_CLUSTER: $KEEP_CLUSTER"
echo "SKIP_BUILD: $SKIP_BUILD"
echo "REUSE_CLUSTER: $REUSE_CLUSTER"
echo "KIND_IMAGE: $KIND_IMAGE"
echo "CLUSTER: $CLUSTER"

hack::ensure_kind
hack::ensure_kubectl

trap 'cleanup' EXIT

function cleanup() {
    if [ -z "$KEEP_CLUSTER" ]; then
        if $KIND_BIN get clusters | grep $CLUSTER; then
            echo "info: deleting the cluster '$CLUSTER'"
            $KIND_BIN delete cluster --name $CLUSTER
        fi
    fi
}

if $KIND_BIN get clusters | grep $CLUSTER; then
    if [ -z "$REUSE_CLUSTER" ]; then
        echo "info: deleting the cluster '$CLUSTER'"
        $KIND_BIN delete cluster --name $CLUSTER
    else
        echo "info: reusing existing cluster"
    fi
fi

if ! $KIND_BIN get clusters | grep $CLUSTER; then
    echo "info: creating the cluster '$CLUSTER'"
    $KIND_BIN create cluster --name $CLUSTER --image $KIND_IMAGE --config hack/kindconfig.yaml
fi

export KUBECONFIG="$($KIND_BIN get kubeconfig-path --name="$CLUSTER")"
$KUBECTL_BIN cluster-info

export CONTROLLER_IMAGE=quay.io/cofyc/advanced-statefulset:latest

if [ -z "$SKIP_BUILD" ]; then
	echo "info: building image $CONTROLLER_IMAGE"
	docker build -t $CONTROLLER_IMAGE .
else
	echo "info: skip building images"
fi

echo "info: loading image $CONTROLLER_IMAGE"
$KIND_BIN load docker-image --name $CLUSTER $CONTROLLER_IMAGE

hack/run-e2e.sh --kubectl-path=$KUBECTL_BIN \
	--provider=skeleton \
	--clean-start=true \
	--repo-root=$ROOT \
    "$@"
