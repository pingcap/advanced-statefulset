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

function usage() {
    cat <<'EOF'
This script is entrypoint to run e2e tests.

Usage: hack/e2e.sh [-h] -- [extra test args]

    -h      show this message and exit

Environments:

    SKIP_BUILD          skip building binaries/images
    SKIP_UP             skip starting the cluster
    SKIP_DOWN           skip shutting down the cluster
    SKIP_TEST           skip running the test
    REUSE_CLUSTER       reuse previous cluster in up phase
    KUBE_VERSION        the version of Kubernetes to test against
    KUBE_WORKERS        the number of worker nodes (excludes master nodes), defaults: 1
    DOCKER_IO_MIRROR    configure mirror for docker.io
    GINKGO_NODES        ginkgo nodes to run specs, defaults: 1
    GINKGO_PARALLEL     if set to `y`, will run specs in parallel, the number of nodes will be the number of cpus
    GINKGO_NO_COLOR     if set to `y`, suppress color output in default reporter

Examples:


0) view help

    ./hack/e2e.sh -h

1) run all specs

    ./hack/e2e.sh
    GINKGO_NODES=8 ./hack/e2e.sh # in parallel

2) limit specs to run

    ./hack/e2e.sh -- --ginkgo.focus='Basic'

    See https://onsi.github.io/ginkgo/ for more ginkgo options.

EOF

}

while getopts "h?" opt; do
    case "$opt" in
    h|\?)
        usage
        exit 0
        ;;
    esac
done

if [ "${1:-}" == "--" ]; then
    shift
fi

SKIP_BUILD=${SKIP_BUILD:-}
SKIP_UP=${SKIP_UP:-}
SKIP_DOWN=${SKIP_DOWN:-}
SKIP_TEST=${SKIP_TEST:-}
REUSE_CLUSTER=${REUSE_CLUSTER:-}
KUBE_VERSION=${KUBE_VERSION:-v1.25}
KUBECONFIG=${KUBECONFIG:-~/.kube/config}
CLUSTER=${CLUSTER:-advanced-statefulset}
DOCKER_IO_MIRROR=${DOCKER_IO_MIRROR:-}
KUBE_WORKERS=${KUBE_WORKERS:-1}

echo "SKIP_UP: $SKIP_UP"
echo "SKIP_DOWN: $SKIP_DOWN"
echo "SKIP_TEST: $SKIP_TEST"
echo "SKIP_BUILD: $SKIP_BUILD"
echo "REUSE_CLUSTER: $REUSE_CLUSTER"
echo "KUBE_VERSION: $KUBE_VERSION"
echo "CLUSTER: $CLUSTER"
echo "DOCKER_IO_MIRROR: $DOCKER_IO_MIRROR"
echo "KUBE_WORKERS: $KUBE_WORKERS"

declare -A kind_node_images
kind_node_images["v1.21.14"]="kindest/node:v1.21.14@sha256:8a4e9bb3f415d2bb81629ce33ef9c76ba514c14d707f9797a01e3216376ba093"
kind_node_images["v1.22.17"]="kindest/node:v1.22.17@sha256:f5b2e5698c6c9d6d0adc419c0deae21a425c07d81bbf3b6a6834042f25d4fba2"
kind_node_images["v1.23.17"]="kindest/node:v1.23.17@sha256:59c989ff8a517a93127d4a536e7014d28e235fb3529d9fba91b3951d461edfdb"
kind_node_images["v1.24.15"]="kindest/node:v1.24.15@sha256:7db4f8bea3e14b82d12e044e25e34bd53754b7f2b0e9d56df21774e6f66a70ab" # no image for v1.24.17 yet
kind_node_images["v1.25.11"]="kindest/node:v1.25.11@sha256:227fa11ce74ea76a0474eeefb84cb75d8dad1b08638371ecf0e86259b35be0c8"

hack::ensure_kind
hack::ensure_kubectl

function e2e::cluster_exists() {
    local name="$1"
    $KIND_BIN get clusters | grep $CLUSTER &>/dev/null
}

function e2e::down() {
    if [ -n "$SKIP_DOWN" ]; then
        echo "info: skip shutting down the cluster '$CLUSTER'"
        return
    fi
    if $KIND_BIN get clusters | grep $CLUSTER &>/dev/null; then
        echo "info: deleting the cluster '$CLUSTER'"
        $KIND_BIN delete cluster --name $CLUSTER
    fi
}

function e2e::__restart_docker() {
    echo "info: restarting docker"
    service docker restart
    # the service can be started but the docker socket not ready, wait for ready
    local WAIT_N=0
    local MAX_WAIT=5
    while true; do
        # docker ps -q should only work if the daemon is ready
        docker ps -q > /dev/null 2>&1 && break
        if [[ ${WAIT_N} -lt ${MAX_WAIT} ]]; then
            WAIT_N=$((WAIT_N+1))
            echo "info; Waiting for docker to be ready, sleeping for ${WAIT_N} seconds."
            sleep ${WAIT_N}
        else
            echo "info: Reached maximum attempts, not waiting any longer..."
            break
        fi
    done
    echo "info: done restarting docker"
}

function e2e::up() {
    if [ -n "$SKIP_UP" ]; then
        echo "info: up phase is skipped"
        return
    fi

    if e2e::cluster_exists $CLUSTER; then
        if [ -z "$REUSE_CLUSTER" ]; then
            echo "info: deleting the cluster '$CLUSTER'"
            $KIND_BIN delete cluster --name $CLUSTER
        else
            echo "info: reusing existing cluster"
        fi
    fi

    if e2e::cluster_exists $CLUSTER; then
        return
    fi

    echo "info: creating the cluster '$CLUSTER'"

    if [ -n "$DOCKER_IO_MIRROR" -a -n "${DOCKER_IN_DOCKER_ENABLED:-}" ]; then
        echo "info: configure docker.io mirror '$DOCKER_IO_MIRROR' for DinD"
cat <<EOF > /etc/docker/daemon.json
{
    "registry-mirrors": ["$DOCKER_IO_MIRROR"]
}
EOF
        e2e::__restart_docker
    fi
    local image=""
    for v in ${!kind_node_images[*]}; do
        if [[ "$KUBE_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ && "$KUBE_VERSION" == "$v" ]]; then
            image=${kind_node_images[$v]}
            echo "info: image for $KUBE_VERSION: $image"
        elif [[ "$KUBE_VERSION" =~ ^v[0-9]+\.[0-9]+$ && "$KUBE_VERSION" == "${v%.*}" ]]; then
            image=${kind_node_images[$v]}
            echo "info: image for $KUBE_VERSION: $image"
        fi
    done
    if [ -z "$image" ]; then
        echo "error: no image for $KUBE_VERSION, exit"
        exit 1
    fi
    if [ -z "$image" ]; then
        echo "error: no image for $KUBE_VERSION, exit"
        exit 1
    fi
    local tmpfile=$(mktemp)
    trap "test -f $tmpfile && rm $tmpfile" RETURN
    cat <<EOF > $tmpfile
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
EOF
    if [ -n "$DOCKER_IO_MIRROR" ]; then
cat <<EOF >> $tmpfile
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
    endpoint = ["$DOCKER_IO_MIRROR"]
EOF
    fi
    # control-plane
    cat <<EOF >> $tmpfile
nodes:
- role: control-plane
EOF
    # workers
    for ((i = 1; i <= $KUBE_WORKERS; i++)) {
        cat <<EOF >> $tmpfile
- role: worker
EOF
    }

    # kubeadm config patches
    # CustomResourceDefaulting feature gate is removed in https://github.com/kubernetes/kubernetes/pull/87475.
    cat <<EOF >> $tmpfile
kubeadmConfigPatches:
- |
  apiVersion: kubeadm.k8s.io/v1beta2
  kind: ClusterConfiguration
  metadata:
    name: config
  apiServer:
    extraArgs:
      "v": "4"
  scheduler:
    extraArgs:
      "v": "4"
  controllerManager:
    extraArgs:
      "v": "4"
- |
  apiVersion: kubeadm.k8s.io/v1beta2
  kind: InitConfiguration
  metadata:
    name: config
  nodeRegistration:
    kubeletExtraArgs:
      "v": "4"
EOF

    # Retry on error. Sometimes, kind will fail with the following error:
    #
    # OCI runtime create failed: container_linux.go:346: starting container process caused "process_linux.go:319: getting the final child's pid from pipe caused \"EOF\"": unknown
    #
    # TODO this error should be related to docker or linux kernel, find the root cause.
    hack::wait_for_success 120 5 "$KIND_BIN create cluster --config $KUBECONFIG --name $CLUSTER --image $image --config $tmpfile -v 4"
}

function e2e::test() {
    if [ -n "$SKIP_TEST" ]; then
        echo "info: test phase is skipped"
        return
    fi
    echo "info: loading image $CONTROLLER_IMAGE"
    $KIND_BIN load docker-image --name $CLUSTER $CONTROLLER_IMAGE

    hack/run-e2e.sh --kubectl-path=$KUBECTL_BIN \
        --kubeconfig=$KUBECONFIG \
        --context=kind-$CLUSTER \
        --provider=skeleton \
        --clean-start=true \
        --delete-namespace-on-failure=false \
        --repo-root=$ROOT \
        "$@"
}

function e2e::build() {
	if [ -n "$SKIP_BUILD" ]; then
        echo "info: build phase is skipped"
        return
	fi
	echo "info: building image $CONTROLLER_IMAGE"
	docker build -t $CONTROLLER_IMAGE .
}

export CONTROLLER_IMAGE=pingcap/advanced-statefulset:latest

trap 'e2e::down' EXIT
e2e::up
e2e::build
e2e::test "$@"
