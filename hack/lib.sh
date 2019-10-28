#!/bin/bash

if [ -z "$ROOT" ]; then
    echo "error: ROOT should be initialized"
    exit 1
fi

OS=$(go env GOOS)
ARCH=$(go env GOARCH)
OUTPUT=${ROOT}/output
OUTPUT_BIN=${OUTPUT}/bin/${OS}
ETCD_VERSION=${ETCD_VERSION:-3.3.17}
KIND_VERSION=0.5.1
KIND_BIN=$OUTPUT_BIN/kind
KUBECTL_VERSION=1.16.0
KUBECTL_BIN=$OUTPUT_BIN/kubectl

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
        [[ "$($KIND_BIN --version | cut -d ' ' -f 3)" == "v$KIND_VERSION" ]]
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
    curl -Lo $tmpfile https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-$(uname)-amd64
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
