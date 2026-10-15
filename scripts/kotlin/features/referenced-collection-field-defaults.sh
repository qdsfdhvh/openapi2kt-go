#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-collection-field-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedCollectionFieldDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import json
import sys
p = Path(sys.argv[1]); checked = sys.argv[3] == 'true'; ref = '#/components/schemas/'
d = json.loads(Path('tests/fixtures/http/referenced-collection-field-defaults.json').read_text())
s = d['components']['schemas']
if checked:
    s['Required']['minItems'] = 1
    s['Required']['items'].update(minItems=1, uniqueItems=True)
    s['Required']['items']['items']['minimum'] = 0
    s['Rows']['maxItems'] = 3
    s['NullableLongs']['maxItems'] = 3
    s['ListMaps']['items']['maxProperties'] = 3
    s['ListMaps']['items']['additionalProperties']['minLength'] = 1
    s['InnerFloatMap']['maxProperties'] = 2
    s['BuiltinFloat']['minimum'] = 1
    s['PatchRows']['minItems'] = 1
    s['PatchRows']['items']['minItems'] = 1
    s['PatchRows']['items']['items']['minimum'] = 0
(p/'input.json').write_text(json.dumps(d))
field = {'$ref': ref+'Values', 'default': {'b': {'b': None}, 'a': {'b': 1.25, 'a': 1}}}
model = {'properties': {'value': field}}
values = {'additionalProperties': {'additionalProperties': {'type': 'number', 'format': 'float', 'nullable': True}}}
if checked:
    values['additionalProperties']['additionalProperties']['minimum'] = 0
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': model}}}}}} for name in ['alpha', 'beta']}
for mode in ['options', 'dedup', 'merge']:
    schemas = {'Values': values}
    if mode == 'merge': schemas.update(Alpha=model, Zeta=model)
    (p/f'{mode}.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': schemas}, 'paths': paths}))
# Independent probes: 240 shallow fields, source48, checked16 and 32 KiB.
limit_schemas = {'Rows': {'type': 'array', 'items': {'type': 'integer'}}, 'Hops': {'properties': {}}}
if checked:
    limit_schemas['Rows'].update(minItems=1, uniqueItems=True)
    limit_schemas['Rows']['items']['minimum'] = 0
limit_schemas['Wide'] = {'properties': {f'v{i}': {'$ref': ref+'Rows', 'default': [1]} for i in range(240)}}
for i in range(45):
    limit_schemas['Hops']['properties'][f'h{i}'] = {'$ref': ref+('SourceRows' if i == 0 else f'Hops/properties/h{i-1}'), 'nullable': False}
limit_schemas['SourceRows'] = {'type': 'array', 'items': {'type': 'string'}}
limit_schemas['Source48'] = {'properties': {'value': {'$ref': ref+'Hops/properties/h44', 'default': []}}}
leaf = {'type': 'integer', **({'minimum': 0} if checked else {})}
for _ in range(16): leaf = {'type': 'array', 'items': leaf}
limit_schemas['DeepRows'] = leaf
limit_schemas['Deep'] = {'properties': {'value': {'$ref': ref+'DeepRows', 'default': []}}}
limit_schemas['Large'] = {'properties': {'value': {'$ref': ref+'SourceRows', 'default': ['a'*32764]}}}
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': limit_schemas}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.referenceddefaults.models.HolderDto(required = listOf(listOf(2L)))
 check(value.copy(kotlin = "copy").rows == value.rows)
'''
if sys.argv[2] == 'data_class' and checked:
    copy += '''
 check(runCatching { value.copy(required = listOf(emptyList())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(mapMaps = mapOf("a" to mapOf("b" to 0.0f))) }.exceptionOrNull() is IllegalArgumentException)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val expected = mapOf("a" to mapOf("a" to 1.0f, "b" to 1.25f), "b" to mapOf("b" to null))
 val optimized = org.example.refoptions.models.ResponseDto()
 val actual: org.example.refoptions.models.ValuesDto? = optimized.value
 check(actual == expected)
 val alias = org.example.refoptions.api.models.AlphaResponseDto()
 check(Json.decodeFromString(org.example.refoptions.models.ResponseDto.serializer(), "{}") == alias)
 val dedup = org.example.refdedup.api.models.BetaResponseDto()
 check(dedup.value == expected)
 check(Json.decodeFromString(org.example.refdedup.api.models.AlphaResponseDto.serializer(), "{}") == dedup)
 val merged = org.example.refmerge.models.ZetaDto()
 check(merged.value == expected && org.example.refmerge.api.models.AlphaResponseDto() == merged)
 check(Json.decodeFromString(org.example.refmerge.models.AlphaDto.serializer(), "{}") == merged)
 check(org.example.refresources.models.WideDto().v239 == listOf(1))
 check(org.example.refresources.models.Source48Dto().value?.isEmpty() == true)
 check(org.example.refresources.models.DeepDto().value?.isEmpty() == true)
 check(org.example.refresources.models.LargeDto().value?.single()?.length == 32764)
''' + copy + ' referencedCollectionFieldDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runReferencedCollectionFieldDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.referenceddefaults --api-client-name ReferencedDefaults --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.refoptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    for mode in dedup merge resources; do
      optimization=()
      if [[ "$mode" == dedup ]]; then optimization=(--deduplicate-models); fi
      if [[ "$mode" == merge ]]; then optimization=(--merge-identical-models); fi
      if [[ "$checked" == true ]]; then optimization+=(--validation); fi
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "org.example.ref$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED REFERENCED COLLECTION FIELD DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedCollectionFieldDefaultsContracts
    echo "FRESH REFERENCED COLLECTION FIELD DEFAULTS $style checked=$checked PASS: $project"
  done
done
