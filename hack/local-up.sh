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

make cmd/controller-manager
export KUBECONFIG=/home/vagrant/.kube/kind-config-advanced-statefulset
kubectl -n kube-system delete ep advanced-statefulset-controller --ignore-not-found
./output/bin/linux/amd64/cmd/controller-manager --kubeconfig $KUBECONFIG -v 4 --leader-elect-resource-name advanced-statefulset-controller  --leader-elect-resource-namespace kube-system
