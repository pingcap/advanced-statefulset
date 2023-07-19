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

if [ -z "$ROOT" ]; then
    echo "error: ROOT should be initialized"
    exit 1
fi

OS=$(go env GOOS)
ARCH=$(go env GOARCH)
OUTPUT=${ROOT}/output
OUTPUT_BIN=${OUTPUT}/bin
ETCD_VERSION=${ETCD_VERSION:-3.5.9}
K8S_VERSION=${K8S_VERSION:-0.24.15}
KIND_VERSION=0.11.1
KIND_BIN=$OUTPUT_BIN/kind
KUBECTL_VERSION=1.24.15
KUBECTL_BIN=$OUTPUT_BIN/kubectl
GINKGO_VERSION=1.11.0
GINKGO_BIN=$OUTPUT_BIN/ginkgo
JQ_VERSION=1.6
JQ_BIN=$OUTPUT_BIN/jq

test -d "$OUTPUT_BIN" || mkdir -p "$OUTPUT_BIN"

hack::download_file() {
  local -r url=$1
  local -r destination_file=$2

  rm "${destination_file}" 2&> /dev/null || true

  for i in $(seq 5)
  do
    if ! curl -fsSL --retry 3 --keepalive-time 2 "${url}" -o "${destination_file}"; then
      echo "Downloading ${url} failed. $((5-i)) retries left."
      sleep 1
    else
      echo "Downloading ${url} succeed"
      return 0
    fi
  done
  return 1
}

hack::install_etcd() {
  (
    local os
    local arch

    os=$(go env GOOS)
    arch=$(go env GOARCH)

	cd output || return 1
    if [[ $(readlink etcd) == etcd-v${ETCD_VERSION}-${os}-* ]]; then
      echo "info: etcd v${ETCD_VERSION} already installed. To use:"
      echo "info: export PATH=\"$(pwd)/etcd:\${PATH}\""
      return
    fi

    if [[ ${os} == "darwin" ]]; then
      download_file="etcd-v${ETCD_VERSION}-darwin-amd64.zip"
      url="https://github.com/coreos/etcd/releases/download/v${ETCD_VERSION}/${download_file}"
      hack::download_file "${url}" "${download_file}"
      unzip -o "${download_file}"
      ln -fns "etcd-v${ETCD_VERSION}-darwin-amd64" etcd
      rm "${download_file}"
    else
      url="https://github.com/coreos/etcd/releases/download/v${ETCD_VERSION}/etcd-v${ETCD_VERSION}-linux-${arch}.tar.gz"
      download_file="etcd-v${ETCD_VERSION}-linux-${arch}.tar.gz"
      hack::download_file "${url}" "${download_file}"
      tar xzf "${download_file}"
      ln -fns "etcd-v${ETCD_VERSION}-linux-${arch}" etcd
      rm "${download_file}"
    fi
    echo "info: etcd v${ETCD_VERSION} installed. To use:"
    echo "info: export PATH=\"$(pwd)/etcd:\${PATH}\""
  )
}

function hack::verify_kind() {
    if test -x "$KIND_BIN"; then
        [[ "$($KIND_BIN --version 2>&1 | cut -d ' ' -f 3)" == "$KIND_VERSION" ]]
        return
    fi
    return 1
}

function hack::ensure_kind() {
    if hack::verify_kind; then
        return 0
    fi
    echo "Installing kind v$KIND_VERSION..."
    tmpfile=$(mktemp)
    trap "test -f $tmpfile && rm $tmpfile" RETURN
    curl -Lo $tmpfile https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-${OS}-${ARCH}
    mv $tmpfile $KIND_BIN
    chmod +x $KIND_BIN
}

function hack::verify_kubectl() {
    if test -x "$KUBECTL_BIN"; then
        [[ "$($KUBECTL_BIN version --client --short | grep -o -P '\d+\.\d+\.\d+')" == "$KUBECTL_VERSION" ]]
        return
    fi
    return 1
}

function hack::ensure_kubectl() {
    if hack::verify_kubectl; then
        return 0
    fi
    echo "Installing kubectl v$KUBECTL_VERSION..."
    tmpfile=$(mktemp)
    trap "test -f $tmpfile && rm $tmpfile" RETURN
    curl -Lo $tmpfile https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/${OS}/${ARCH}/kubectl
    mv $tmpfile $KUBECTL_BIN
    chmod +x $KUBECTL_BIN
}

function hack::verify_ginkgo() {
    if test -x "$GINKGO_BIN"; then
        [[ "$($GINKGO_BIN version | grep -o -P '\d+\.\d+\.\d+')" == "$GINKGO_VERSION" ]]
        return
    fi
    return 1
}

function hack::ensure_ginkgo() {
    if hack::verify_ginkgo; then
        return 0
    fi
    echo "Installing ginkgo v$GINKGO_VERSION..."
    GOBIN=$OUTPUT_BIN go install github.com/onsi/ginkgo/ginkgo@v$GINKGO_VERSION
    if ! hack::verify_ginkgo; then
        echo "info: installed ginkgo ($GINKGO_BIN) does not match expected version $GINKGO_VERSION"
        exit 1
    fi
}

function hack::verify_jq() {
    if test -x "$JQ_BIN"; then
        [[ "$($JQ_BIN --version | awk -F- '{print $2}' )" == "$JQ_VERSION" ]]
        return
    fi
    return 1
}

function hack::ensure_jq() {
    if hack::verify_jq; then
        return 0
    fi
    echo "Installing jq v$JQ_VERSION..."
    if [[ "$OS" == "linux" ]]; then
        hack::download_file https://github.com/stedolan/jq/releases/download/jq-$JQ_VERSION/jq-linux64 $JQ_BIN
    elif [[ "$OS" == "darwin" ]]; then
        hack::download_file https://github.com/stedolan/jq/releases/download/jq-$JQ_VERSION/jq-osx-amd64 $JQ_BIN
    fi
    chmod +x $JQ_BIN
}

# hack::version_ge "$v1" "$v2" checks whether "v1" is greater or equal to "v2"
function hack::version_ge() {
    [ "$(printf '%s\n' "$1" "$2" | sort -V | head -n1)" = "$2" ]
}

# Usage:
#
#	hack::wait_for_success 120 5 "cmd arg1 arg2 ... argn"
#
# Returns 0 if the shell command get output, 1 otherwise.
# From https://github.com/kubernetes/kubernetes/blob/v1.17.0/hack/lib/util.sh#L70
function hack::wait_for_success() {
    local wait_time="$1"
    local sleep_time="$2"
    local cmd="$3"
    while [ "$wait_time" -gt 0 ]; do
        if eval "$cmd"; then
            return 0
        else
            sleep "$sleep_time"
            wait_time=$((wait_time-sleep_time))
        fi
    done
    return 1
}

# Reads in stdin and adds it line by line to the array provided. This can be
# used instead of "mapfile -t", and is bash 3 compatible.
#
# Assumed vars:
#   $1 (name of array to create/modify)
#
# Example usage:
# hack::read-array files < <(ls -1)
#
function hack::read-array {
	local i=0
	unset -v "$1"
	while IFS= read -r "$1[i++]"; do :; done
	eval "[[ \${$1[--i]} ]]" || unset "$1[i]" # ensures last element isn't empty
}

function hack::ensure_openapi() {
    echo "Installing openpi_gen..."
    GOBIN=$OUTPUT_BIN go install k8s.io/code-generator/cmd/openapi-gen@v$K8S_VERSION
}
