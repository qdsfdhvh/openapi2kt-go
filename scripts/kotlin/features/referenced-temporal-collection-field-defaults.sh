#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-temporal-collection-field-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedTemporalCollectionFieldDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import json
import sys
p = Path(sys.argv[1]); checked = sys.argv[3] == 'true'; ref = '#/components/schemas/'
d = json.loads(Path('tests/fixtures/http/referenced-temporal-collection-field-defaults.json').read_text())
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
    s['Days']['maxItems'] = 3
    s['Stamps']['maxProperties'] = 3
    s['RequiredDays']['minItems'] = 1
    s['RequiredDays']['items']['minItems'] = 1
    s['Sources']['properties']['temporal']['maxItems'] = 2
    s['DateListMaps']['items']['maxProperties'] = 2
    s['InstantMapLists']['additionalProperties']['maxItems'] = 2
    s['InstantMapMaps']['maxProperties'] = 2
(p/'input.json').write_text(json.dumps(d))
field = {'$ref': ref+'Values', 'default': {'b': {'b': None}, 'a': {'b': '2024-02-29T12:34:56+02:30', 'a': '2024-02-29T00:00:00Z'}}}
model = {'properties': {'value': field}}
values = {'additionalProperties': {'additionalProperties': {'$ref': ref+'Stamp'}}}
if checked:
    values['additionalProperties']['maxProperties'] = 2
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': model}}}}}} for name in ['alpha', 'beta']}
for mode in ['options', 'dedup', 'merge']:
    schemas = {'Values': values, 'Stamp': {'type': 'string', 'format': 'date-time', 'nullable': True}}
    if mode == 'merge': schemas.update(Alpha=model, Zeta=model)
    (p/f'{mode}.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': schemas}, 'paths': paths}))
# Independent probes: 240 shallow fields, source48, checked16 and 32 KiB.
limit_schemas = {'Rows': {'type': 'array', 'items': {'type': 'string', 'format': 'date'}}, 'Hops': {'properties': {}}}
if checked:
    limit_schemas['Rows'].update(minItems=1)
limit_schemas['Wide'] = {'properties': {f'v{i}': {'$ref': ref+'Rows', 'default': ['2024-02-29']} for i in range(240)}}
for i in range(45):
    limit_schemas['Hops']['properties'][f'h{i}'] = {'$ref': ref+('SourceRows' if i == 0 else f'Hops/properties/h{i-1}'), 'nullable': False}
limit_schemas['SourceRows'] = {'type': 'array', 'items': {'type': 'string', 'format': 'date'}}
limit_schemas['TextRows'] = {'type': 'array', 'items': {'type': 'string'}}
limit_schemas['Source48'] = {'properties': {'value': {'$ref': ref+'Hops/properties/h44', 'default': []}}}
leaf = {'type': 'string', 'format': 'date'}
for _ in range(16): leaf = {'type': 'array', 'items': leaf, **({'minItems': 0} if checked else {})}
limit_schemas['DeepRows'] = leaf
limit_schemas['Deep'] = {'properties': {'value': {'$ref': ref+'DeepRows', 'default': []}}}
limit_schemas['Large'] = {'properties': {'value': {'$ref': ref+'TextRows', 'default': ['a'*32764]}}}
limit_schemas['DateMapping'] = {'additionalProperties': {'type': 'string', 'format': 'date'}}
limit_schemas['LargeTemporal'] = {'properties': {'value': {'$ref': ref+'DateMapping', 'default': {'a'*32751: '2024-02-29'}}}}
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': limit_schemas}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.referencedtemporaldefaults.models.HolderDto(requiredDays = listOf(listOf(kotlinx.datetime.LocalDate.parse("2024-02-29"))), required = listOf(listOf(2L)))
 check(value.copy(kotlin = "copy", kotlinx = "copy").rows == value.rows)
 check(value.copy(namedDateList = listOf(LocalDate.parse("2024-03-01"))).namedDateList == listOf(LocalDate.parse("2024-03-01")))
'''
if sys.argv[2] == 'data_class' and checked:
    copy += '''
 check(runCatching { value.copy(required = listOf(emptyList())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(requiredDays = listOf(emptyList())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(namedDateList = List(4) { LocalDate.parse("2024-02-29") }) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(mapMaps = mapOf("a" to mapOf("b" to 0.0f))) }.exceptionOrNull() is IllegalArgumentException)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class, kotlin.time.ExperimentalTime::class)
package org.example.contracts
import kotlin.time.Instant
import kotlinx.datetime.LocalDate
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val expected = mapOf("a" to mapOf("a" to Instant.parse("2024-02-29T00:00:00Z"), "b" to Instant.parse("2024-02-29T12:34:56+02:30")), "b" to mapOf("b" to null))
 val optimized = org.example.refoptions.models.ResponseDto()
 val actual: org.example.refoptions.models.ValuesDto? = optimized.value
 val backing: Map<String, Map<String, Instant?>>? = actual
 check(backing == expected)
 check(org.example.refoptions.models.ResponseDto.serializer().descriptor.serialName == "org.example.refoptions.models.ResponseDto")
 check(org.example.refdedup.api.models.BetaResponseDto.serializer().descriptor.serialName == "org.example.refdedup.api.models.AlphaResponseDto")
 check(org.example.refmerge.models.ZetaDto.serializer().descriptor.serialName == "org.example.refmerge.models.AlphaDto")
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
   val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
   check(json.decodeFromString(org.example.refoptions.models.ResponseDto.serializer(), "{}") == optimized)
 }
 check(actual == expected)
 val alias = org.example.refoptions.api.models.AlphaResponseDto()
 check(Json.decodeFromString(org.example.refoptions.models.ResponseDto.serializer(), "{}") == alias)
 val dedup = org.example.refdedup.api.models.BetaResponseDto()
 check(dedup.value == expected)
 check(Json.decodeFromString(org.example.refdedup.api.models.AlphaResponseDto.serializer(), "{}") == dedup)
 val merged = org.example.refmerge.models.ZetaDto()
 check(merged.value == expected && org.example.refmerge.api.models.AlphaResponseDto() == merged)
 check(Json.decodeFromString(org.example.refmerge.models.AlphaDto.serializer(), "{}") == merged)
 check(org.example.refresources.models.WideDto().v239 == listOf(LocalDate.parse("2024-02-29")))
 check(org.example.refresources.models.Source48Dto().value?.isEmpty() == true)
 check(org.example.refresources.models.DeepDto().value?.isEmpty() == true)
 check(org.example.refresources.models.LargeDto().value?.single()?.length == 32764)
 val largeTemporal = org.example.refresources.models.LargeTemporalDto().value
 check(largeTemporal?.keys?.single()?.length == 32751 && largeTemporal?.values?.single() == LocalDate.parse("2024-02-29"))
''' + copy + ' referencedTemporalCollectionFieldDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runReferencedTemporalCollectionFieldDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.referencedtemporaldefaults --api-client-name ReferencedTemporalDefaults --model-name-suffix Dto --model-style "$style")
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
      echo "PREPARED REFERENCED TEMPORAL COLLECTION FIELD DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedTemporalCollectionFieldDefaultsContracts
    echo "FRESH REFERENCED TEMPORAL COLLECTION FIELD DEFAULTS $style checked=$checked PASS: $project"
  done
done
