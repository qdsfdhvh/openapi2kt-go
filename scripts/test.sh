#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/kotlin/common.sh"
features=(tuple-special-references tuple-recursive-cover tuple-hierarchy-references tuple-product-union-sources tuple-union-sources tuple-structured-const tuple-scalar-const tuple-null-sources tuple-collection-sources tuple-inline-allof tuple-boolean-references tuple-inline-dto tuple-boolean-members tuple-reference-defaults tuple-inline-enums tuple-raw-defaults tuple-boolean-tails form-field-defaults narrowed-enum-wrapper-defaults implicit-string-enum-null discarded-structured-const pinned-structured-const false-dto-owner-alias-defaults false-dto-owner-root-defaults literal-false-collection-dto-field-defaults literal-false-collection-root-defaults false-generic-signatures literal-false-collection-defaults literal-false-collection-fields literal-false-dto-field-defaults literal-false-root-defaults false-nominal-signatures literal-false-fields false-inline-collection-defaults false-inline-collection-fields false-dto-independent-defaults false-dto-field-counts false-dto-field-defaults false-object-default-members false-reference-field-defaults false-alias-defaults false-collection-defaults false-collection-references false-reference-fields false-component-roots nullable-recursive-cover-edges tuple-native-enum-references allof-default-overrides dto-allof-unique dto-collection-reference-leaves-unique dto-string-collection-reference-unique dto-native-collection-reference-unique dto-special-collection-reference-unique dto-builtin-collection-reference-unique dto-string-reference-unique dto-native-reference-unique dto-scalar-reference-unique dto-string-collection-unique dto-native-collection-unique null-component-factories dto-null-collection-unique null-value-defaults null-const-object-defaults dto-collection-unique dto-null-unique dto-const-unique dto-enum-unique dto-unique allof-directional-models composed-target-owners allof-target-intersections composed-leaf-intersections hierarchy-dto-intersections composed-dto-intersections internal-dto-intersections nullable-bridge-intersections bridge-intersections jvm-copy-signatures jvm-signature-limits allof-uuid-defaults standard-form-fields wide-deep-validation enum-form-fields floating-form-fields optional-form-fields tuple-root-defaults allof-numeric-defaults unformatted-native-factories object-collection-factories allof-string-defaults composed-member-defaults tuple-field-defaults raw-request-media tuple-target-defaults tuple-allof-references form-request-media tuple-dto-references merge-primitive-unions tuple-enum-references split-locked-adoption tuple-references inline-error-roots object-default-graphs map-unique uuid-unique floating-unique native-enum-unique string-enum-unique temporal-unique nullable-composed-reference-members parameter-styles referenced-bytes-field-defaults raw-response-media directional-models yaml-document http-namespace-collisions http-servers allof-nominal-terminals boolean-http boolean-root-nullable boolean-root boolean-value collection-owned-dto composed-nested-owners hierarchy-dispatch hierarchy-fields hierarchy-intrinsic-nullable hierarchy-nested-owners hierarchy-null-members hierarchy-nullable-alias hierarchy-reference-chains internal-dto-chain internal-dto merge named-dto-alias-chain named-dto-chain primitive-tuple recursive-json root-dto-bridge root-internal-dto union-allof union-nominal-owners enum-nominal-owners valueclass-reference-chains composed-union-terminals temporal-reference-chains builtin-reference-chains root-borrowed-builtin bigint-reference-chains bytes-reference-chains caller-native-reference-chains raw-true-reference-chains recursive-cover-reference-chains named-collection-reference-chains root-borrowed-collection borrowed-collection-dto-leaves borrowed-collection-enum-leaves borrowed-collection-valueclass-leaves borrowed-collection-allof-model-leaves borrowed-collection-union-leaves borrowed-collection-product-union-leaves borrowed-collection-hierarchy-leaves borrowed-collection-builtin-leaves borrowed-collection-temporal-leaves borrowed-collection-bigint-leaves borrowed-collection-bytes-leaves borrowed-collection-caller-native-leaves borrowed-collection-raw-true-leaves borrowed-collection-recursive-cover-leaves union-common-dictionaries nullable-allof-dto union-common-nominals union-inline-values nullable-product-unions nullable-object-members nullable-object-reference-members model-defaults enum-defaults json-unique inline-errors merge-exclusions native-enum-validation native-enum-consumers native-enum-bounds internal-native-enums named-native-enums nullable-native-enums native-enum-null-members string-enum-null-members referenced-allof-bounds referenced-allof-counts explicit-null-literals inferred-null-literals null-only-defaults structured-const-fields structured-const-values nullable-structured-const primitive-array-defaults checked-array-defaults primitive-map-defaults native-enum-collection-defaults string-enum-collection-defaults nested-builtin-collection-defaults nested-temporal-collection-defaults nested-string-enum-collection-defaults nested-native-enum-collection-defaults referenced-collection-field-defaults string-enum-collection-bounds referenced-scalar-field-defaults referenced-temporal-scalar-field-defaults referenced-temporal-collection-field-defaults referenced-string-enum-collection-defaults referenced-native-enum-collection-defaults referenced-native-enum-scalar-defaults referenced-string-enum-scalar-defaults referenced-string-enum-null-member-defaults nested-object-defaults nested-object-collection-defaults referenced-object-defaults referenced-object-collection-defaults union-field-defaults raw-json-field-defaults direct-nullable-component-counts bytes-field-defaults bytes-component-defaults reference-wrapper-component-defaults hierarchy-inline-members nullable-component-defaults raw-json-component-defaults)
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
