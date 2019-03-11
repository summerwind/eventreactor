#!/bin/bash

set -e

docker run -it --rm \
  -v ${PWD}:/go/src/github.com/summerwind/eventreactor \
  -w /go/src/github.com/summerwind/eventreactor \
  summerwind/kubebuilder:latest \
  make release
