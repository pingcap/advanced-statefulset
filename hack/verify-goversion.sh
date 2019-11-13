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

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

if [[ -z "$(command -v go)" ]]; then
	tee <<EOF
Can't find 'go' in PATH, please fix and retry.
See http://golang.org/doc/install for installation instructions.
EOF
	exit 1
fi  

expected_go_version=go$(cat .go-version)

IFS=" " read -ra go_version <<< "$(go version)"
if [[ "${expected_go_version}" != "${go_version[2]}" ]]; then
    tee <<EOF
Detected go version: ${go_version[*]}.
Please install ${expected_go_version}.
EOF
	exit 2
fi 
