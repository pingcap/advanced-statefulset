#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

make cmd/controller-manager
export KUBECONFIG=/home/vagrant/.kube/kind-config-advanced-statefulset
kubectl -n kube-system delete ep advanced-statefulset-controller --ignore-not-found
./output/bin/linux/amd64/cmd/controller-manager --kubeconfig $KUBECONFIG -v 4 --leader-elect-resource-name advanced-statefulset-controller  --leader-elect-resource-namespace kube-system
