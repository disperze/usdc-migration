#!/usr/bin/env bash
# Regenerates gogoproto + grpc-gateway code into x/usdcmigration/types.
set -euo pipefail

# Resolves deps from the pinned buf.lock; run `buf dep update` in proto/
# deliberately when bumping them.
cd "$(dirname "$0")/../proto"
buf generate --template buf.gen.gogo.yaml

cd ..
# buf writes to the go_package path; move it into the module tree.
cp -r github.com/giansalex/usdc-migration/v8/* ./
rm -rf github.com
