#!/bin/bash

set -e

if [ ! -e ca.pem ]; then
  echo "Generating CA certificate files..."
  cfssl gencert -initca ca-csr.json | cfssljson -bare ca
fi

echo "Generating server certificate files..."
cfssl gencert -ca=ca.pem -ca-key=ca-key.pem server-csr.json | cfssljson -bare server
