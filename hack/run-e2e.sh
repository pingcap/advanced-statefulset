#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

TEST_ARGS=(
    test
    -timeout=60m
    -v
    ./test/e2e
)

TEST_ARGS+=("$@")

if [ -n "$KUBECONFIG" ]; then
    TEST_ARGS+=("-kubeconfig=$KUBECONFIG")
fi

echo "Running e2e tests:" >&2
echo "go ${TEST_ARGS[@]}" >&2
exec "go" "${TEST_ARGS[@]}"
