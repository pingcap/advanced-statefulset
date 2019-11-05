#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

test_args=(
    -timeout=360m
    -v
    ./test/e2e
)

test_args+=("$@")

echo "Running e2e tests:" >&2
echo "go test ${test_args[@]}" >&2
exec go test "${test_args[@]}"
