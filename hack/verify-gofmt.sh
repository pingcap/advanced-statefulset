#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

gofmt=$(which gofmt)

PKGS=$(go list ./... | grep -v /vendor/)

# gofmt exits with non-zero exit code if it finds a problem unrelated to
# formatting (e.g., a file does not parse correctly). Without "|| true" this
# would have led to no useful error message from gofmt, because the script would
# have failed before getting to the "echo" in the block below.
diff=$(xargs -n 1 -I pkg echo ${GOPATH}/src/pkg <<<"${PKGS}" | xargs ${gofmt} -d -s 2>&1) || true
if [[ -n "${diff}" ]]; then
    echo "${diff}"
    exit 1
else
    echo "gofmt check passed."
fi
