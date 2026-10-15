#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
: "${OPENAPI_KT_BINARY:?Provide the independently built native Go CLI}"
binary="$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$OPENAPI_KT_BINARY")"
mkdir -p .pi/go-migration
workspace="$(mktemp -d .pi/go-migration/kotlin-cli-run-XXXXXX)"
export_root="$root/$workspace/project"

# Go generation/export and Kotlin compilation are offline. The opt-in exporter
# checks native Go build identity, actual CLI reports/manifest hashes, and
# check/no-op mtimes against the independent in-memory API for every source.
env -u GOROOT GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
  OPENAPI_KT_TEST_EXPORT="$export_root" \
  OPENAPI_KT_TEST_EXPORT_HIERARCHY=1 OPENAPI_KT_TEST_BINARY="$binary" \
  "${GO:-go}" test -count=1 -v ./core -run '^TestExportModelSmoke$'

if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
  printf 'PREPARED native Go CLI Kotlin project: %s\n' "$export_root"
  exit 0
fi

"${GRADLE:-gradle}" --offline --no-daemon -p "$export_root" \
  compileKotlin runEnumContracts runPrimitiveContracts \
  runNullableCompositionContracts runConstContracts runObjectCompositionContracts \
  runEnumContextContracts runScalarPortContracts runPrimitiveUnionContracts \
  runUnionNormalizationContracts runRootPrimitivePortContracts \
  runObjectUnionContracts runRootUnionContracts runRootKindContracts \
  runScalarProjectionContracts runHierarchyContracts runHierarchyPortContracts \
  runDictionaryContracts "$@"
printf 'Fresh native Go CLI Kotlin project: %s\n' "$export_root"
