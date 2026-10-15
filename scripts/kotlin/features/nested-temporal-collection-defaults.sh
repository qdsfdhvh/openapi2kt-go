#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/nested-temporal-collection-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedTemporalCollectionDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import json
import sys
p = Path(sys.argv[1]); checked = sys.argv[3] == 'true'
d = json.loads(Path('tests/fixtures/http/nested-temporal-collection-defaults.json').read_text())
if checked:
    fields = d['components']['schemas']['Holder']['properties']
    fields['required']['minItems'] = 1
    fields['required']['items']['minItems'] = 1
    fields['rows']['maxItems'] = 3
    fields['rows']['items']['maxItems'] = 3
    fields['listMaps']['items']['maxProperties'] = 3
    fields['mapLists']['additionalProperties']['maxItems'] = 2
    fields['mapMaps']['additionalProperties']['maxProperties'] = 3
    fields['builtin']['additionalProperties']['additionalProperties']['minimum'] = 0
    patch = d['paths']['/patch']['patch']['requestBody']['content']['application/json']['schema']['properties']
    patch['rows']['minItems'] = 1
    patch['rows']['items']['minItems'] = 1
    patch['stamps']['additionalProperties']['minItems'] = 1
(p/'input.json').write_text(json.dumps(d))
field = {'additionalProperties': {'additionalProperties': {'type': 'string', 'format': 'date-time', 'nullable': True}}, 'default': {'b': {'b': None}, 'a': {'b': '2024-02-29T12:34:56+02:30', 'a': '2024-02-29T00:00:00Z'}}}
if checked:
    field['additionalProperties']['maxProperties'] = 2
schema = {'properties': {'value': field}}
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': schema}}}}}} for name in ['alpha', 'beta']}
for mode in ['options', 'dedup', 'merge']:
    document = {'openapi': '3.1.0', 'paths': paths}
    if mode == 'merge': document['components'] = {'schemas': {'Alpha': schema, 'Zeta': schema}}
    (p/f'{mode}.json').write_text(json.dumps(document))
# Independent source48 / checked16 / slots240 / default32KiB probes.
wide = {'properties': {f'v{i}': {'type': 'array', 'items': {'type': 'array', 'items': {'type': 'string', 'format': 'date'}}, 'default': [['2024-02-29']]} for i in range(240)}}
leaf = {'type': 'string', 'format': 'date-time'}
for _ in range(16 if checked else 47):
    leaf = {'type': 'array', 'items': leaf, **({'minItems': 0} if checked else {})}
leaf['default'] = []
large = {'type': 'array', 'items': {'additionalProperties': {'type': 'string', 'format': 'date'}}, 'default': [{'a'*32749: '2024-02-29'}]}
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'Wide': wide, 'Deep': {'properties': {'value': leaf}}, 'Large': {'properties': {'value': large}}}}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.nestedtemporal.models.HolderDto(required = listOf(listOf(LocalDate.parse("2024-02-29"))))
 check(value.copy(kotlin = "copy").rows == value.rows)
'''
if sys.argv[2] == 'data_class' and checked:
    copy += '''
 check(runCatching { value.copy(required = listOf(emptyList())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(mapMaps = mapOf("a" to mapOf("a" to null, "b" to null, "c" to null, "d" to null))) }.exceptionOrNull() is IllegalArgumentException)
'''
if checked:
    copy += '''
 val bad = mapOf("x" to mapOf("a" to null, "b" to null, "c" to null))
 check(runCatching { org.example.temporaloptions.api.models.BetaResponseDto(value = bad) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { org.example.temporaldedup.api.models.BetaResponseDto(value = bad) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { org.example.temporalmerge.models.ZetaDto(value = bad) }.exceptionOrNull() is IllegalArgumentException)
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
   val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
   val invalid = """{"value":{"x":{"a":null,"b":null,"c":null}}}"""
   val optionFailure = runCatching { json.decodeFromString(org.example.temporaloptions.api.models.BetaResponseDto.serializer(), invalid) }.exceptionOrNull()
   val dedupFailure = runCatching { json.decodeFromString(org.example.temporaldedup.api.models.BetaResponseDto.serializer(), invalid) }.exceptionOrNull()
   val mergeFailure = runCatching { json.decodeFromString(org.example.temporalmerge.models.ZetaDto.serializer(), invalid) }.exceptionOrNull()
   for (failure in listOf(optionFailure, dedupFailure, mergeFailure)) {
     check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message.orEmpty().contains("maxProperties") })
   }
 }
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlin.time.Instant
import kotlinx.coroutines.runBlocking
import kotlinx.datetime.LocalDate
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val expected = mapOf("a" to mapOf("a" to Instant.parse("2024-02-29T00:00:00Z"), "b" to Instant.parse("2024-02-29T10:04:56Z")), "b" to mapOf("b" to null))
 val optimized = org.example.temporaloptions.models.ResponseDto()
 val actual: Map<String, Map<String, Instant?>>? = optimized.value
 check(actual == expected)
 val alias = org.example.temporaloptions.api.models.AlphaResponseDto()
 check(alias::class == optimized::class)
 check(org.example.temporaloptions.api.models.AlphaResponseDto.serializer().descriptor.serialName == "org.example.temporaloptions.models.ResponseDto")
 check(Json.decodeFromString(org.example.temporaloptions.models.ResponseDto.serializer(), "{}") == alias)
 val dedup = org.example.temporaldedup.api.models.BetaResponseDto()
 check(dedup.value == expected && dedup::class == org.example.temporaldedup.api.models.AlphaResponseDto()::class)
 check(org.example.temporaldedup.api.models.BetaResponseDto.serializer().descriptor.serialName == "org.example.temporaldedup.api.models.AlphaResponseDto")
 val merged = org.example.temporalmerge.models.ZetaDto()
 check(merged.value == expected && org.example.temporalmerge.api.models.AlphaResponseDto() == merged)
 check(org.example.temporalmerge.models.ZetaDto.serializer().descriptor.serialName == "org.example.temporalmerge.models.AlphaDto")
 check(Json.decodeFromString(org.example.temporalmerge.models.AlphaDto.serializer(), "{}") == merged)
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
   val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
   check(json.decodeFromString(org.example.temporaloptions.api.models.BetaResponseDto.serializer(), "{}") == optimized)
   check(json.decodeFromString(org.example.temporaldedup.api.models.BetaResponseDto.serializer(), "{}") == dedup)
   check(json.decodeFromString(org.example.temporalmerge.models.ZetaDto.serializer(), "{}") == merged)
 }
 check(org.example.temporalresources.models.WideDto().v239 == listOf(listOf(LocalDate.parse("2024-02-29"))))
 check(org.example.temporalresources.models.DeepDto().value?.isEmpty() == true)
 check(org.example.temporalresources.models.LargeDto().value?.single()?.keys?.single()?.length == 32749)
''' + copy + ' nestedTemporalCollectionDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runNestedTemporalCollectionDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.nestedtemporal --api-client-name NestedTemporal --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.temporaloptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
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
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "org.example.temporal$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED NESTED TEMPORAL COLLECTION DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNestedTemporalCollectionDefaultsContracts
    echo "FRESH NESTED TEMPORAL COLLECTION DEFAULTS $style checked=$checked PASS: $project"
  done
done
