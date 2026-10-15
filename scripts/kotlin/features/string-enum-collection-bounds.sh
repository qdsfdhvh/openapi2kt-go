#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
prepare_only=false
if [[ "${1:-}" == --prepare-only && $# -eq 1 ]]; then prepare_only=true
elif [[ $# -ne 0 ]]; then printf 'Usage: bash scripts/kotlin/features/string-enum-collection-bounds.sh [--prepare-only]\n' >&2; exit 2; fi
mkdir -p build
workspace="$(mktemp -d "$root/build/string-enum-collection-bounds-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/StringEnumCollectionBoundsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/string-enum-collection-bounds.json "$project/bounded.json"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import copy
import json
import sys
p = Path(sys.argv[1])
checked = sys.argv[3] == 'true'
d = json.loads((p/'bounded.json').read_text())
# Validation-off is an accepted unbounded control with the same lexical shape.
# Bounds in the original source must still reject without validation.
def remove_checks(v):
    if isinstance(v, dict):
        for key in ['minLength', 'maxLength', 'pattern', 'minItems', 'maxItems', 'minProperties', 'maxProperties']:
            v.pop(key, None)
        for child in v.values():
            remove_checks(child)
    elif isinstance(v, list):
        for child in v:
            remove_checks(child)
if not checked:
    remove_checks(d)
(p/'input.json').write_text(json.dumps(d))
schema = {'properties': {
    'codes': {'type': 'array', 'items': {'enum': ['one', 'two', 'x']}, 'default': ['one']},
    'labels': {'additionalProperties': {'enum': ['yes', 'no']}, 'default': {'b': 'no', 'a': 'yes'}}}}
if checked:
    schema['properties']['codes']['items'].update(minLength=2, maxLength=6, pattern='^[a-z]+$')
    schema['properties']['codes']['maxItems'] = 2
    schema['properties']['labels']['additionalProperties'].update(minLength=2, maxLength=6)
bounds = copy.deepcopy(schema)
if checked:
    bounds['properties']['codes']['items']['minLength'] = 3
else:
    bounds['properties']['codes']['items']['enum'].append('separate')
other_default = copy.deepcopy(schema)
other_default['properties']['codes']['default'] = ['two']
def operation(name, shape):
    return {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': shape}}}}}}
paths = {f'/{name}': operation(name, shape) for name, shape in [('alpha', schema), ('beta', schema), ('bounds', bounds), ('defaults', other_default)]}
(p/'options.json').write_text(json.dumps({'openapi': '3.1.0', 'paths': paths}))
(p/'dedup.json').write_text(json.dumps({'openapi': '3.1.0', 'paths': paths}))
(p/'merge.json').write_text(json.dumps({'openapi': '3.1.0', 'paths': paths, 'components': {'schemas': {'Alpha': schema, 'Zeta': schema}}}))
# Independent compilation probes retain the existing caps/heap. Deep checked
# builtin controls do not grant nested enum trees admission.
wide = {'properties': {f'f{i}': {'type': 'array', 'items': {'enum': ['one'], **({'minLength': 2, 'maxLength': 6, 'pattern': '^[a-z]+$'} if checked else {})}} for i in range(240)}}
(p/'limits.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'Wide': wide}}}))
copy_calls = ''
if sys.argv[2] == 'data_class':
    copy_calls = '''
 val value = org.example.stringenumbounds.models.HolderDto(required = listOf(org.example.stringenumbounds.models.HolderDto.RequiredInner.TWO))
 check(value.copy(kotlin = "copy").labels == value.labels)
 check(value.copy(codes = listOf(org.example.stringenumbounds.models.HolderDto.CodesInner.Unrecognized("future"))).codes?.single()?.serialName == "future")
'''
    if checked:
        copy_calls += '''
 for ((bad, rule) in listOf("x" to "minLength", "toolong" to "maxLength", "BAD" to "pattern", "!" to "minLength", "futurelong" to "maxLength", "FUTURE" to "pattern")) {
  val failure = runCatching { value.copy(codes = listOf(org.example.stringenumbounds.models.HolderDto.CodesInner.fromSerialName(bad))) }.exceptionOrNull()
  check(failure is IllegalArgumentException && failure.message.orEmpty().contains(rule))
  check(runCatching { value.copy(labels = mapOf("a" to org.example.stringenumbounds.models.HolderDto.LabelsValue.fromSerialName(bad))) }.exceptionOrNull() is IllegalArgumentException)
 }
 check(runCatching { value.copy(required = emptyList()) }.exceptionOrNull() is IllegalArgumentException)
'''
main = '''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val optimized = org.example.stringboundsoptions.models.ResponseDto()
 check(optimized.codes?.single() == org.example.stringboundsoptions.models.ResponseDto.CodesInner.ONE)
 check(optimized.labels?.get("a") == org.example.stringboundsoptions.models.ResponseDto.LabelsValue.YES)
 check(org.example.stringboundsoptions.api.models.AlphaResponseDto() == optimized)
 check(org.example.stringboundsoptions.api.models.BetaResponseDto() == optimized)
 check(org.example.stringboundsoptions.models.ResponseDto.CodesInner.serializer().descriptor.serialName == "org.example.stringboundsoptions.models.ResponseDto.CodesInner")
 val dedup = org.example.stringboundsdedup.api.models.BetaResponseDto()
 check(dedup.codes?.single() == org.example.stringboundsdedup.api.models.AlphaResponseDto.CodesInner.ONE)
 val merged = org.example.stringboundsmerge.models.ZetaDto()
 check(merged.labels?.get("a") == org.example.stringboundsmerge.models.AlphaDto.LabelsValue.YES)
 check(org.example.stringboundsmerge.api.models.AlphaResponseDto() == merged)
 check(org.example.stringboundsdedup.api.models.DefaultsResponseDto().codes?.single()?.serialName == "two")
 check(org.example.stringboundsoptions.api.models.DefaultsResponseDto().codes?.single()?.serialName == "two")
 check(org.example.stringboundsoptions.api.models.BoundsResponseDto.CodesInner.ONE.serialName == "one")
 check(org.example.stringboundsdedup.api.models.BoundsResponseDto.CodesInner.ONE.serialName == "one")
 check(org.example.stringboundsmerge.api.models.BoundsResponseDto.CodesInner.ONE.serialName == "one")
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
  val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
  check(json.decodeFromString(org.example.stringboundsoptions.models.ResponseDto.serializer(), "{}") == optimized)
  check(json.decodeFromString(org.example.stringboundsdedup.api.models.AlphaResponseDto.serializer(), "{}") == dedup)
  check(json.decodeFromString(org.example.stringboundsmerge.models.AlphaDto.serializer(), "{}") == merged)
  check(json.decodeFromString(org.example.stringboundsoptions.models.ResponseDto.serializer(), """{"codes":["future"]}""").codes?.single()?.serialName == "future")
'''
if checked:
    main += '''
  for (bad in listOf("x", "FUTURE", "futurelong")) {
   check(runCatching { json.decodeFromString(org.example.stringboundsoptions.models.ResponseDto.serializer(), "{\\"codes\\":[\\"$bad\\"]}") }.isFailure)
   check(runCatching { json.decodeFromString(org.example.stringboundsdedup.api.models.AlphaResponseDto.serializer(), "{\\"codes\\":[\\"$bad\\"]}") }.isFailure)
   check(runCatching { json.decodeFromString(org.example.stringboundsmerge.models.AlphaDto.serializer(), "{\\"codes\\":[\\"$bad\\"]}") }.isFailure)
  }
'''
main += ' }\n' + copy_calls + ' stringEnumCollectionBoundsContracts(' + sys.argv[3] + ') }\n'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(main)
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runStringEnumCollectionBoundsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.stringenumbounds --api-client-name StringEnumBounds --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.stringboundsoptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    # Source with bounds must reject off and preserve the accepted output.
    "$workspace/openapi-kt" generate --input "$project/bounded.json" --output "$project/build/generated" --package-name org.example.stringenumbounds --diagnostic-format json > "$project/rejection.json" && exit 1
    python3 - "$project/rejection.json" <<'PYREJECT'
import json
import sys
d = json.load(open(sys.argv[1]))
assert d['status'] == 'error' and d['files'] == []
assert d['diagnostics'][0]['code'] == 'E_SCHEMA_UNSUPPORTED'
PYREJECT
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    for mode in dedup merge limits; do
      optimization=()
      if [[ "$mode" == dedup ]]; then optimization=(--deduplicate-models); fi
      if [[ "$mode" == merge ]]; then optimization=(--merge-identical-models); fi
      if [[ "$checked" == true ]]; then optimization+=(--validation); fi
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "org.example.stringbounds$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "$prepare_only" == false ]]; then
      "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runStringEnumCollectionBoundsContracts
      echo "FRESH STRING ENUM COLLECTION BOUNDS $style checked=$checked PASS: $project"
    fi
  done
done
printf 'Fresh String enum collection bounds projects (prepare_only=%s): %s\n' "$prepare_only" "$workspace"
