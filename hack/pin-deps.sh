#!/bin/bash

# Pin all k8s.io dependencies to a specified verison.

VERSION=1.16.0

go mod edit -require k8s.io/kubernetes@v$VERSION

STAGING_REPOS=($(curl -sS https://raw.githubusercontent.com/kubernetes/kubernetes/v${VERSION}/go.mod | sed -n 's|.*k8s.io/\(.*\) => ./staging/src/k8s.io/.*|k8s.io/\1|p'))

for repo in ${STAGING_REPOS[@]}; do
    PKG_VERSION_RESOLVED=$(go mod download -json $repo@kubernetes-$VERSION | jq -r '.Version')
    go mod edit -replace $repo=$repo@${PKG_VERSION_RESOLVED}
    go mod tidy
done
