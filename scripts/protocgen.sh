#!/usr/bin/env bash
# Regenerates gogoproto + grpc-gateway code into x/usdcmigration/types.
set -euo pipefail

cd "$(dirname "$0")/../proto"
buf dep update
buf generate --template buf.gen.gogo.yaml

cd ..
# buf writes to the go_package path; move it into the module tree.
cp -r github.com/giansalex/usdc-migration/v8/* ./
rm -rf github.com
