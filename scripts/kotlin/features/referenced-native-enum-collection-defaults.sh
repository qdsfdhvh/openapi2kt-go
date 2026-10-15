#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-native-enum-collection-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedNativeEnumCollectionDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import json
import sys
p = Path(sys.argv[1]); checked = sys.argv[3] == 'true'; ref = '#/components/schemas/'
d = json.loads(Path('tests/fixtures/http/referenced-native-enum-collection-defaults.json').read_text())
s = d['components']['schemas']
if checked:
    # Counts and retained numeric bounds join validation; the scalar wire and
    # original vocabulary membership stay exactly as without validation.
    s['Required']['minItems'] = 1
    s['Required']['items']['minItems'] = 1
    s['Ints']['maxItems'] = 3
    s['Int32Code']['minimum'] = 1
    s['Int32Code']['maximum'] = 3
    s['Longs']['maxItems'] = 3
    s['DoubleMaps']['maxProperties'] = 3
    s['DoubleCode']['minimum'] = -1
    s['IntMaps']['maxProperties'] = 3
    s['RequiredInts']['minItems'] = 1
    s['Nil']['minProperties'] = 1
    s['NestedInts']['maxItems'] = 3
    s['PatchRows']['items']['minItems'] = 1
(p/'input.json').write_text(json.dumps(d))
field = {'$ref': ref+'Values', 'default': {'b': [2], 'a': [1, None]}}
model = {'properties': {'value': field}}
values = {'additionalProperties': {'type': 'array', 'items': {'$ref': ref+'Code', 'nullable': True}}}
if checked:
    values['additionalProperties']['maxItems'] = 2
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': model}}}}}} for name in ['alpha', 'beta']}
for mode in ['options', 'dedup', 'merge']:
    schemas = {'Values': values, 'Code': {'type': 'integer', 'format': 'int32', 'enum': [1, 2]}}
    if mode == 'merge': schemas.update(Alpha=model, Zeta=model)
    (p/f'{mode}.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': schemas}, 'paths': paths}))
# Independent probes: 240 shallow fields, source48, checked16 and 32 KiB.
limit_schemas = {'Code': {'type': 'integer', 'format': 'int32', 'enum': [1, 2]}, 'Hops': {'properties': {}}, 'CodeRows': {'type': 'array', 'items': {'type': 'integer', 'format': 'int32', 'enum': [1, 2]}}}
if checked:
    limit_schemas['CodeRows']['minItems'] = 1
limit_schemas['Wide'] = {'properties': {f'v{i}': {'$ref': ref+'CodeRows', 'default': [2]} for i in range(240)}}
for i in range(45):
    limit_schemas['Hops']['properties'][f'h{i}'] = {'$ref': ref+('SourceRows' if i == 0 else f'Hops/properties/h{i-1}'), 'nullable': False}
limit_schemas['SourceRows'] = {'type': 'array', 'items': {'type': 'integer', 'format': 'int32', 'enum': [1, 2]}}
limit_schemas['Source48'] = {'properties': {'value': {'$ref': ref+'Hops/properties/h44', 'default': []}}}
leaf = {'type': 'integer', 'format': 'int32', 'enum': [1, 2]}
for _ in range(16): leaf = {'type': 'array', 'items': leaf, **({'minItems': 0} if checked else {})}
limit_schemas['DeepRows'] = leaf
limit_schemas['Deep'] = {'properties': {'value': {'$ref': ref+'DeepRows', 'default': []}}}
limit_schemas['CodeMapping'] = {'additionalProperties': {'$ref': ref+'Code'}}
limit_schemas['Large'] = {'properties': {'value': {'$ref': ref+'CodeMapping', 'default': {'a'*32753: 2}}}}
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': limit_schemas}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.refnativedefaults.models.HolderDto(required = listOf(listOf(1)), requiredInts = listOf(3))
 check(value.copy(kotlin = "copy", kotlinx = "copy", companion = "copy", serializer = "copy").ints == value.ints)
 check(value.copy(required = listOf(listOf(2))).required?.single()?.single() == 2)
 check(value.copy(floats = listOf(1.0f)).floats == listOf(1.0f))
'''
if sys.argv[2] == 'data_class' and checked:
    copy += '''
 check(runCatching { value.copy(ints = List(4) { 1 }) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(ints = listOf(9)) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(requiredInts = emptyList()) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(required = listOf(listOf(9))) }.exceptionOrNull() is IllegalArgumentException)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val expected = mapOf("a" to listOf(1, null), "b" to listOf(2))
 val optimized = org.example.refnativeoptions.models.ResponseDto()
 val actual: org.example.refnativeoptions.models.ValuesDto? = optimized.value
 check(actual == expected)
 check(org.example.refnativeoptions.models.ResponseDto.serializer().descriptor.serialName == "org.example.refnativeoptions.models.ResponseDto")
 check(org.example.refnativededup.api.models.BetaResponseDto.serializer().descriptor.serialName == "org.example.refnativededup.api.models.AlphaResponseDto")
 check(org.example.refnativemerge.models.ZetaDto.serializer().descriptor.serialName == "org.example.refnativemerge.models.AlphaDto")
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
   val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
   check(json.decodeFromString(org.example.refnativeoptions.models.ResponseDto.serializer(), "{}") == optimized)
 }
 check(actual == expected)
 val alias = org.example.refnativeoptions.api.models.AlphaResponseDto()
 check(Json.decodeFromString(org.example.refnativeoptions.models.ResponseDto.serializer(), "{}") == alias)
 val dedup = org.example.refnativededup.api.models.BetaResponseDto()
 check(dedup.value == mapOf("a" to listOf(1, null), "b" to listOf(2)))
 check(Json.decodeFromString(org.example.refnativededup.api.models.AlphaResponseDto.serializer(), "{}") == dedup)
 val merged = org.example.refnativemerge.models.ZetaDto()
 check(merged.value == mapOf("a" to listOf(1, null), "b" to listOf(2)) && org.example.refnativemerge.api.models.AlphaResponseDto() == merged)
 check(Json.decodeFromString(org.example.refnativemerge.models.AlphaDto.serializer(), "{}") == merged)
 check(org.example.refnativeresources.models.WideDto().v239 == listOf(2))
 check(org.example.refnativeresources.models.Source48Dto().value?.isEmpty() == true)
 check(org.example.refnativeresources.models.DeepDto().value?.isEmpty() == true)
 val large = org.example.refnativeresources.models.LargeDto().value
 check(large?.keys?.single()?.length == 32753 && large?.values?.single() == 2)
''' + copy + ' referencedNativeEnumCollectionFieldDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runReferencedNativeEnumCollectionDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.refnativedefaults --api-client-name RefNativeDefaults --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.refnativeoptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
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
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "org.example.refnative$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED REFERENCED NATIVE ENUM COLLECTION FIELD DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedNativeEnumCollectionDefaultsContracts
    echo "FRESH REFERENCED NATIVE ENUM COLLECTION FIELD DEFAULTS $style checked=$checked PASS: $project"
  done
done
