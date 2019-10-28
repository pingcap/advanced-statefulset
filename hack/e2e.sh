#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

KEEP_CLUSTER=${KEEP_CLUSTER:-}
REUSE_CLUSTER=${REUSE_CLUSTER:-}

hack::ensure_kind
hack::ensure_kubectl

CLUSTER=advanced-statefulset

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
    $KIND_BIN create cluster --name $CLUSTER --image kindest/node:v1.16.1 --config hack/kindconfig.yaml
fi

export KUBECONFIG="$($KIND_BIN get kubeconfig-path --name="$CLUSTER")"
$KUBECTL_BIN cluster-info

hack/run-e2e.sh --kubectl-path=$KUBECTL_BIN \
    --ginkgo.focus='\[sig-apps\]\sStatefulSet\s' \
    "$@"
