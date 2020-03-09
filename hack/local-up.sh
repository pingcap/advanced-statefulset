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

CLUSTER=${CLUSTER:-kind}

make cmd/controller-manager

cfgfile=$(mktemp)
trap "test -f $cfgfile && rm $cfgfile" EXIT
kind get kubeconfig --name "$CLUSTER" > $cfgfile

KUBE_VERSION=$(kubectl version --short | awk '/Server Version:/ {print $3}')

if hack::version_ge $KUBE_VERSION "v1.16.0"; then
    kubectl --kubeconfig $cfgfile apply -f manifests/crd.v1.yaml
else
    kubectl --kubeconfig $cfgfile apply -f manifests/crd.v1beta1.yaml
fi

kubectl --kubeconfig $cfgfile -n kube-system delete ep advanced-statefulset-controller --ignore-not-found
./output/bin/linux/amd64/cmd/controller-manager --kubeconfig $cfgfile -v 4 --leader-elect-resource-name advanced-statefulset-controller --leader-elect-resource-namespace kube-system
