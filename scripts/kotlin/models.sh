#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Always use the native Go command. An explicit binary bypasses building, not
# generation/manifest/report verification. Every compiler run uses fresh output.
if [[ -z "${OPENAPI_KT_BINARY:-}" ]]; then
  mkdir -p .pi/go-migration
  workspace="$(mktemp -d .pi/go-migration/cli-build-XXXXXX)"
  openapi_kt_build_cli "$workspace/openapi-kt"
  export OPENAPI_KT_BINARY="$root/$workspace/openapi-kt"
fi
exec bash scripts/kotlin/export-models.sh "$@"
