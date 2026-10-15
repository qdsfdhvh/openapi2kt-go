#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/kotlin/common.sh"
features=(yaml-document http-namespace-collisions http-servers allof-nominal-terminals boolean-http boolean-root-nullable boolean-root boolean-value collection-owned-dto composed-nested-owners hierarchy-dispatch hierarchy-fields hierarchy-intrinsic-nullable hierarchy-nested-owners hierarchy-null-members hierarchy-nullable-alias hierarchy-reference-chains internal-dto-chain internal-dto merge named-dto-alias-chain named-dto-chain primitive-tuple recursive-json root-dto-bridge root-internal-dto union-allof union-nominal-owners enum-nominal-owners valueclass-reference-chains composed-union-terminals temporal-reference-chains builtin-reference-chains root-borrowed-builtin bigint-reference-chains bytes-reference-chains caller-native-reference-chains raw-true-reference-chains recursive-cover-reference-chains named-collection-reference-chains root-borrowed-collection borrowed-collection-dto-leaves borrowed-collection-enum-leaves borrowed-collection-valueclass-leaves borrowed-collection-allof-model-leaves borrowed-collection-union-leaves borrowed-collection-product-union-leaves borrowed-collection-hierarchy-leaves borrowed-collection-builtin-leaves borrowed-collection-temporal-leaves borrowed-collection-bigint-leaves borrowed-collection-bytes-leaves borrowed-collection-caller-native-leaves borrowed-collection-raw-true-leaves borrowed-collection-recursive-cover-leaves union-common-dictionaries nullable-allof-dto union-common-nominals union-inline-values nullable-product-unions nullable-object-members nullable-object-reference-members model-defaults enum-defaults json-unique inline-errors merge-exclusions native-enum-validation native-enum-consumers native-enum-bounds internal-native-enums named-native-enums nullable-native-enums native-enum-null-members string-enum-null-members referenced-allof-bounds referenced-allof-counts explicit-null-literals inferred-null-literals null-only-defaults structured-const-fields structured-const-values nullable-structured-const primitive-array-defaults checked-array-defaults primitive-map-defaults native-enum-collection-defaults string-enum-collection-defaults nested-builtin-collection-defaults nested-temporal-collection-defaults nested-string-enum-collection-defaults nested-native-enum-collection-defaults referenced-collection-field-defaults string-enum-collection-bounds referenced-scalar-field-defaults referenced-temporal-scalar-field-defaults referenced-temporal-collection-field-defaults referenced-string-enum-collection-defaults referenced-native-enum-collection-defaults referenced-native-enum-scalar-defaults referenced-string-enum-scalar-defaults referenced-string-enum-null-member-defaults nested-object-defaults nested-object-collection-defaults referenced-object-defaults referenced-object-collection-defaults union-field-defaults raw-json-field-defaults direct-nullable-component-counts)

usage() {
  printf 'Usage: bash scripts/test.sh {unit|models|http|features|feature NAME|all|--list}\n'
}

unit() {
  local formatting
  printf '\n== gofmt check ==\n'
  formatting="$("${GOFMT:-gofmt}" -l core internal cmd)"
  if [[ -n "$formatting" ]]; then printf '%s\n' "$formatting"; return 1; fi
  for command in 'vet' 'test' 'race'; do
    case "$command" in
      vet) printf '\n== go vet ./... ==\n'; env -u GOROOT GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off "${GO:-go}" vet ./... ;;
      test) printf '\n== go test ./... ==\n'; env -u GOROOT GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off "${GO:-go}" test ./... ;;
      race) printf '\n== go test -race ./... ==\n'; env -u GOROOT GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off "${GO:-go}" test -race ./... ;;
    esac
  done
  printf '\n== Python tooling tests ==\n'
  local test_bash="$BASH"
  if command -v cygpath >/dev/null 2>&1; then
    test_bash="$(cygpath -w "$test_bash")"
  fi
  OPENAPI_KT_TEST_BASH="${OPENAPI_KT_TEST_BASH:-$test_bash}" \
    "${PYTHON:-python3}" -m unittest discover -s scripts/tests -p 'test_*.py'
}

run_features() {
  local name
  for name in "${features[@]}"; do bash "scripts/kotlin/features/$name.sh"; done
}

if [[ $# -eq 0 ]]; then usage >&2; exit 2; fi
command="$1"; shift
case "$command" in
  --help|-h) usage ;;
  --list) printf 'unit\nmodels\nhttp\nfeatures\nall\n'; printf 'feature %s\n' "${features[@]}" ;;
  models) exec bash scripts/kotlin/models.sh "$@" ;;
  http) exec bash scripts/kotlin/http.sh "$@" ;;
  feature)
    if [[ $# -ne 1 ]]; then usage >&2; exit 2; fi
    for name in "${features[@]}"; do
      if [[ "$1" == "$name" ]]; then exec bash "scripts/kotlin/features/$name.sh"; fi
    done
    printf 'Unknown feature. Use --list for supported names.\n' >&2; exit 2 ;;
  unit|features|all)
    if [[ $# -ne 0 ]]; then usage >&2; exit 2; fi
    case "$command" in
      unit) unit ;;
      features) run_features ;;
      all) unit; bash scripts/kotlin/models.sh; bash scripts/kotlin/http.sh; run_features ;;
    esac ;;
  *) usage >&2; exit 2 ;;
esac
