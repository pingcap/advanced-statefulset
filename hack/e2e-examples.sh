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

#
# E2E entrypoint script for examples.
#

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

source "${ROOT}/hack/lib.sh"

hack::ensure_kind
hack::ensure_kubectl
hack::ensure_jq

export PATH=$OUTPUT_BIN:$PATH

echo "info: create a Kubernetes cluster"
kind create cluster

logfile=$(mktemp)

function cleanup() {
    echo "info: cleaning up"
	echo "info: deleting the cluster"
    kind delete cluster

	echo "info: killing the controller manager"
    [[ -n "${CONTROLLER_PID:-}" ]] && hack::read-array CONTROLLER_PIDS < <(pgrep -P "${CONTROLLER_PID}" ; ps -o pid= -p "${CONTROLLER_PID}")
    [[ -n "${CONTROLLER_PIDS:-}" ]] && sudo kill "${CONTROLLER_PIDS[@]}" 2>/dev/null

	if ! test -f $logfile; then
        return
	fi

    echo "info: logs of controller manager"
    cat $logfile
    rm $logfile
}

trap "cleanup" EXIT

echo "info: start advanced statefulset controller manager"
hack/local-up.sh &> $logfile &
CONTROLLER_PID=$!

echo "info: testing examples"

function sts_is_ready() {
    local name="$1"
    local desiredReplicas=$(kubectl get asts $name -ojsonpath='{.spec.replicas}')
	desiredReplicas=${desiredReplicas:-0}
    local readyReplicas=$(kubectl get asts $name -ojsonpath='{.status.readyReplicas}')
	readyReplicas=${readyReplicas:-0}
	if [[ "$desiredReplicas" -eq 0 ]]; then
		echo "got 'desiredReplicas' desired replicas, expects > 0 value"
		return 1
	fi
    if [[ "$readyReplicas" != "$desiredReplicas" ]]; then
        echo "got '$readyReplicas' ready replicas, expects '$desiredReplicas'"
        return 1
    fi
    echo "got '$readyReplicas' ready replicas, expects '$desiredReplicas', sts is ready"
    return 0
}

function crd_is_ready() {
    local name="$1"
    local established=$(kubectl get crd statefulsets.apps.pingcap.com -o json | jq '.status["conditions"][] | select(.type == "Established") | .status')
    if [ $? -ne 0 ]; then
        return 1
    fi
    [[ "$established" == "True" ]]
}

hack::wait_for_success 100 3 "crd_is_ready statefulsets.apps.pingcap.com"

# after `crd_is_ready`, kubectl apply may still fail with
# `error: unable to recognize "examples/statefulset.yaml": no matches for kind "StatefulSet" in version "apps.pingcap.com/v1alpha1"`
# so we need to wait for a while
sleep 10

kubectl apply -f examples/statefulset.yaml
hack::wait_for_success 100 3 "sts_is_ready web"
