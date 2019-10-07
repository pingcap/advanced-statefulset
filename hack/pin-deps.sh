#!/bin/bash
#
# Pin all k8s.io dependencies to a specified version.
#

VERSION=1.16.0

go mod edit -require k8s.io/kubernetes@v$VERSION

STAGING_REPOS=($(curl -sS https://raw.githubusercontent.com/kubernetes/kubernetes/v${VERSION}/go.mod | sed -n 's|.*k8s.io/\(.*\) => ./staging/src/k8s.io/.*|k8s.io/\1|p'))

edit_args=()
for repo in ${STAGING_REPOS[@]}; do
    edit_args+=(-replace $repo=$repo@kubernetes-1.16.0)
done

go mod edit ${edit_args[@]}
go mod tidy
