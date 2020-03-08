#!/bin/bash

# Copyright 2020 PingCAP, Inc.
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

# This script is used to generate version informations from git repo automatically.
#
# Output format:
#
#   <KEY1>: <VALUE1>
#   <KEY2>: <VALUE2>
#
# You can simply use this command to extrac the value by key:
#
#   ./hack/version.sh | awk -F' ' '/^GIT_VERSION:/ {print $2}'
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

GIT_EXACT_TAG=$(git describe --tags --abbrev=0 --exact-match 2>/dev/null || true)
GIT_RECENT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || true)
GIT_SHA_SHORT=$(git rev-parse --short HEAD)
GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

echo "GIT_EXACT_TAG: $GIT_EXACT_TAG"
echo "GIT_RECENT_TAG: $GIT_RECENT_TAG"
echo "GIT_DIRTY: $GIT_DIRTY"
echo "GIT_SHA_SHORT: $GIT_SHA_SHORT"

# Modifed from k8s.io/kubernetes/hack/lib/version.sh.
if [[ -n ${GIT_VERSION-} ]] || GIT_VERSION=$(git describe --tags --abbrev=7 HEAD 2>/dev/null); then
    # This translates the "git describe" to an actual semver.org
    # compatible semantic version that looks something like this:
    #   v1.1.0-alpha.0.6+84c76d1
    DASHES_IN_GIT_VERSION=$(echo "${GIT_VERSION}" | sed "s/[^-]//g")
    if [[ "${DASHES_IN_GIT_VERSION}" == "---" ]] ; then
        # We have distance to subversion (v1.1.0-subversion-1-gCommitHash)
        GIT_VERSION=$(echo "${GIT_VERSION}" | sed "s/-\([0-9]\{1,\}\)-g\([0-9a-f]\{7\}\)$/.\1\+\2/")
    elif [[ "${DASHES_IN_GIT_VERSION}" == "--" ]] ; then
        # We have distance to base tag (v1.1.0-1-gCommitHash)
        GIT_VERSION=$(echo "${GIT_VERSION}" | sed "s/-g\([0-9a-f]\{7\}\)$/+\1/")
    fi
    if [[ "$GIT_DIRTY" == "dirty" ]]; then
        # git describe --dirty only considers changes to existing files, but
        # that is problematic since new untracked .go files affect the build,
        # so use our idea of "dirty" from git status instead.
        GIT_VERSION+="-dirty"
    fi
fi
echo "GIT_VERSION: $GIT_VERSION"
