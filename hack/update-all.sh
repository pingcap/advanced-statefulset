#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

hack/update-vendor.sh
hack/update-codegen.sh
hack/update-gofmt.sh
