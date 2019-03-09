#!/bin/bash

set -e

BASE_DIR="$(cd $(dirname $0)/../ && pwd)/release"
TARGET_OS="linux darwin"

mkdir -p ${BASE_DIR}
cd ${BASE_DIR}

echo "Generating manifests..."
kustomize build ../config/default > eventreactor.yaml
kustomize build ../config/addons > eventreactor-addons.yaml

echo "Building binaries..."
for os in ${TARGET_OS}; do
  CGO_ENABLED=0 GOOS=${os} go build -o reactorctl github.com/summerwind/eventreactor/cmd/reactorctl
  tar zcf reactorctl-${os}-amd64.tar.gz reactorctl
  rm -rf reactorctl
done
