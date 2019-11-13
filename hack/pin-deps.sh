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

VERSION=1.16.0

go mod edit -require k8s.io/kubernetes@v$VERSION

STAGING_REPOS=($(curl -sS https://raw.githubusercontent.com/kubernetes/kubernetes/v${VERSION}/go.mod | sed -n 's|.*k8s.io/\(.*\) => ./staging/src/k8s.io/.*|k8s.io/\1|p'))

edit_args=()
for repo in ${STAGING_REPOS[@]}; do
    edit_args+=(-replace $repo=$repo@kubernetes-$VERSION)
done

go mod edit ${edit_args[@]}
go mod tidy
