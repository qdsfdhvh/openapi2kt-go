#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-string-enum-collection-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedStringEnumCollectionDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import json
import sys
p = Path(sys.argv[1]); checked = sys.argv[3] == 'true'; ref = '#/components/schemas/'
d = json.loads(Path('tests/fixtures/http/referenced-string-enum-collection-defaults.json').read_text())
s = d['components']['schemas']
if checked:
    # Counts stay checked; enum vocabulary keeps its open runtime and String
    # bounds/pattern never join this profile.
    s['Required']['minItems'] = 1
    s['Required']['items']['minItems'] = 1
    s['Rows']['maxItems'] = 3
    s['NullableRows']['maxItems'] = 3
    s['ListMaps']['maxItems'] = 3
    s['ListMaps']['items']['maxProperties'] = 3
    s['MapLists']['maxProperties'] = 3
    s['MapLists']['additionalProperties']['maxItems'] = 3
    s['MapMaps']['maxProperties'] = 2
    s['RequiredModes']['minItems'] = 1
    s['NullableModes']['minItems'] = 1
(p/'input.json').write_text(json.dumps(d))
field = {'$ref': ref+'Values', 'default': {'b': ['off'], 'a': ['on', None]}}
model = {'properties': {'value': field}}
values = {'additionalProperties': {'type': 'array', 'items': {'$ref': ref+'Status', 'nullable': True}}}
if checked:
    values['additionalProperties']['maxItems'] = 2
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': model}}}}}} for name in ['alpha', 'beta']}
for mode in ['options', 'dedup', 'merge']:
    schemas = {'Values': values, 'Status': {'type': 'string', 'enum': ['on', 'off']}}
    if mode == 'merge': schemas.update(Alpha=model, Zeta=model)
    (p/f'{mode}.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': schemas}, 'paths': paths}))
# Independent probes: 240 shallow fields, source48, checked16 and 32 KiB.
limit_schemas = {'Status': {'type': 'string', 'enum': ['on', 'off']}, 'Hops': {'properties': {}}, 'StatusRows': {'type': 'array', 'items': {'$ref': ref+'Status'}}}
if checked:
    limit_schemas['StatusRows']['minItems'] = 1
limit_schemas['Wide'] = {'properties': {f'v{i}': {'$ref': ref+'StatusRows', 'default': ['on']} for i in range(240)}}
for i in range(45):
    limit_schemas['Hops']['properties'][f'h{i}'] = {'$ref': ref+('SourceRows' if i == 0 else f'Hops/properties/h{i-1}'), 'nullable': False}
limit_schemas['SourceRows'] = {'type': 'array', 'items': {'type': 'string', 'enum': ['on', 'off']}}
limit_schemas['Source48'] = {'properties': {'value': {'$ref': ref+'Hops/properties/h44', 'default': []}}}
leaf = {'type': 'string', 'enum': ['on', 'off']}
for _ in range(16): leaf = {'type': 'array', 'items': leaf, **({'minItems': 0} if checked else {})}
limit_schemas['DeepRows'] = leaf
limit_schemas['Deep'] = {'properties': {'value': {'$ref': ref+'DeepRows', 'default': []}}}
limit_schemas['StatusMapping'] = {'additionalProperties': {'$ref': ref+'Status'}}
limit_schemas['Large'] = {'properties': {'value': {'$ref': ref+'StatusMapping', 'default': {'a'*32751: 'on'}}}}
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': limit_schemas}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.refstringdefaults.models.HolderDto(required = listOf(listOf(org.example.refstringdefaults.models.StatusDto.ON)), requiredModes = listOf(org.example.refstringdefaults.models.NullableStatusDto.OFF))
 check(value.copy(kotlin = "copy", kotlinx = "copy", companion = "copy", serializer = "copy").rows == value.rows)
 check(value.copy(required = listOf(listOf(org.example.refstringdefaults.models.StatusDto.OFF))).required?.single()?.single() == org.example.refstringdefaults.models.StatusDto.OFF)
'''
if sys.argv[2] == 'data_class' and checked:
    copy += '''
 check(runCatching { value.copy(rows = List(4) { null }) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(required = listOf(emptyList())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(requiredModes = emptyList()) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(mapLists = mapOf("a" to List(4) { null })) }.exceptionOrNull() is IllegalArgumentException)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val expected = mapOf("a" to listOf(org.example.refstringoptions.models.StatusDto.ON, null), "b" to listOf(org.example.refstringoptions.models.StatusDto.OFF))
 val optimized = org.example.refstringoptions.models.ResponseDto()
 val actual: org.example.refstringoptions.models.ValuesDto? = optimized.value
 check(actual == expected)
 check(org.example.refstringoptions.models.ResponseDto.serializer().descriptor.serialName == "org.example.refstringoptions.models.ResponseDto")
 check(org.example.refstringdedup.api.models.BetaResponseDto.serializer().descriptor.serialName == "org.example.refstringdedup.api.models.AlphaResponseDto")
 check(org.example.refstringmerge.models.ZetaDto.serializer().descriptor.serialName == "org.example.refstringmerge.models.AlphaDto")
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
   val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
   check(json.decodeFromString(org.example.refstringoptions.models.ResponseDto.serializer(), "{}") == optimized)
 }
 check(actual == expected)
 val alias = org.example.refstringoptions.api.models.AlphaResponseDto()
 check(Json.decodeFromString(org.example.refstringoptions.models.ResponseDto.serializer(), "{}") == alias)
 val dedup = org.example.refstringdedup.api.models.BetaResponseDto()
 check(dedup.value == mapOf("a" to listOf(org.example.refstringdedup.models.StatusDto.ON, null), "b" to listOf(org.example.refstringdedup.models.StatusDto.OFF)))
 check(Json.decodeFromString(org.example.refstringdedup.api.models.AlphaResponseDto.serializer(), "{}") == dedup)
 val merged = org.example.refstringmerge.models.ZetaDto()
 check(merged.value == mapOf("a" to listOf(org.example.refstringmerge.models.StatusDto.ON, null), "b" to listOf(org.example.refstringmerge.models.StatusDto.OFF)) && org.example.refstringmerge.api.models.AlphaResponseDto() == merged)
 check(Json.decodeFromString(org.example.refstringmerge.models.AlphaDto.serializer(), "{}") == merged)
 check(org.example.refstringresources.models.WideDto().v239 == listOf(org.example.refstringresources.models.StatusDto.ON))
 check(org.example.refstringresources.models.Source48Dto().value?.isEmpty() == true)
 check(org.example.refstringresources.models.DeepDto().value?.isEmpty() == true)
 val large = org.example.refstringresources.models.LargeDto().value
 check(large?.keys?.single()?.length == 32751 && large?.values?.single() == org.example.refstringresources.models.StatusDto.ON)
''' + copy + ' referencedStringEnumCollectionFieldDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runReferencedStringEnumCollectionDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.refstringdefaults --api-client-name RefStringDefaults --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.refstringoptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
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
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "org.example.refstring$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED REFERENCED STRING ENUM COLLECTION FIELD DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedStringEnumCollectionDefaultsContracts
    echo "FRESH REFERENCED STRING ENUM COLLECTION FIELD DEFAULTS $style checked=$checked PASS: $project"
  done
done
