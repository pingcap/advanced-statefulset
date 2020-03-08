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

#
# Pin all k8s.io dependencies to a specified version.
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

VERSION=1.17.3
STAGING_VERSION=0.17.3 # // since 1.17

go mod edit -require k8s.io/kubernetes@v$VERSION

STAGING_REPOS=($(curl -sS https://raw.githubusercontent.com/kubernetes/kubernetes/v${VERSION}/go.mod | sed -n 's|.*k8s.io/\(.*\) => ./staging/src/k8s.io/.*|k8s.io/\1|p'))

edit_args=()
for repo in ${STAGING_REPOS[@]}; do
    # pre-1.17.0
    # edit_args+=(-replace $repo=$repo@kubernetes-$VERSION)
    edit_args+=(-replace $repo=$repo@v$STAGING_VERSION)
done

go mod edit ${edit_args[@]}
go mod tidy
go mod vendor
