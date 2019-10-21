#!/bin/bash

if [ -z "$ROOT" ]; then
    echo "error: ROOT should be initialized"
    exit 1
fi

cd $ROOT

test -d output || mkdir output

ETCD_VERSION=${ETCD_VERSION:-3.3.17}

hack::host_os() {
  local host_os
  case "$(uname -s)" in
    Darwin)
      host_os=darwin
      ;;
    Linux)
      host_os=linux
      ;;
    *)
      echo "error: Unsupported host OS.  Must be Linux or Mac OS X."
      exit 1
      ;;
  esac
  echo "${host_os}"
}

hack::host_arch() {
  local host_arch
  case "$(uname -m)" in
    x86_64*)
      host_arch=amd64
      ;;
    i?86_64*)
      host_arch=amd64
      ;;
    amd64*)
      host_arch=amd64
      ;;
    aarch64*)
      host_arch=arm64
      ;;
    arm64*)
      host_arch=arm64
      ;;
    arm*)
      host_arch=arm
      ;;
    i?86*)
      host_arch=x86
      ;;
    s390x*)
      host_arch=s390x
      ;;
    ppc64le*)
      host_arch=ppc64le
      ;;
    *)
      echo "error: Unsupported host arch. Must be x86_64, 386, arm, arm64, s390x or ppc64le."
      exit 1
      ;;
  esac
  echo "${host_arch}"
}

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

    os=$(hack::host_os)
    arch=$(hack::host_arch)

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
