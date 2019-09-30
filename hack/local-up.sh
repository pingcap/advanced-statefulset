#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

make cmd/controller-manager
export KUBECONFIG=/home/vagrant/.kube/kind-config-kind
kubectl -n kube-system delete ep advanced-statefulset --ignore-not-found
./output/bin/linux/amd64/cmd/controller-manager --kubeconfig /home/vagrant/.kube/kind-config-kind -v 4
