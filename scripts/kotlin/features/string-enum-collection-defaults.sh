#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/string-enum-collection-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/StringEnumCollectionDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/string-enum-collection-defaults.json "$project/input.json"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import json
import sys
p = Path(sys.argv[1])
checked = sys.argv[3] == 'true'
if checked:
    d = json.loads((p/'input.json').read_text())
    fields = d['components']['schemas']['Holder']['properties']
    fields['required']['minItems'] = 1
    fields['codes']['maxItems'] = 3
    fields['labels']['maxProperties'] = 2
    patch = d['paths']['/patch']['patch']['requestBody']['content']['application/json']['schema']['properties']
    patch['codes']['minItems'] = 1
    (p/'input.json').write_text(json.dumps(d))
# Exercise actual relocated lexical helpers under all owned-shape options.
schema = {'properties': {
    'codes': {'type': 'array', 'items': {'enum': ['one', 'two']}, 'default': ['one']},
    'labels': {'additionalProperties': {'enum': ['yes', 'no']}, 'default': {'b': 'no', 'a': 'yes'}}}}
if checked:
    schema['properties']['codes']['maxItems'] = 2
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': schema}}}}}} for name in ['alpha', 'beta']}
(p/'options.json').write_text(json.dumps({'openapi': '3.1.0', 'paths': paths}))
(p/'dedup.json').write_text(json.dumps({'openapi': '3.1.0', 'paths': paths}))
(p/'merge.json').write_text(json.dumps({'openapi': '3.1.0', 'paths': paths, 'components': {'schemas': {'Alpha': schema, 'Zeta': schema}}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.stringenumcollections.models.HolderDto(required = listOf(org.example.stringenumcollections.models.HolderDto.RequiredInner.TWO))
 check(value.copy(kotlin = "copy").labels == value.labels)
 check(value.copy(codes = listOf(org.example.stringenumcollections.models.HolderDto.CodesInner.Unrecognized("future"))).codes?.single()?.serialName == "future")
'''
if sys.argv[2] == 'data_class' and checked:
    copy += '''
 check(runCatching { value.copy(required = emptyList()) }.exceptionOrNull() is IllegalArgumentException)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val optimized = org.example.stringenumoptions.models.ResponseDto()
 check(optimized.codes?.single() == org.example.stringenumoptions.models.ResponseDto.CodesInner.ONE)
 check(optimized.labels?.get("a") == org.example.stringenumoptions.models.ResponseDto.LabelsValue.YES)
 val alias = org.example.stringenumoptions.api.models.AlphaResponseDto()
 check(Json.decodeFromString(org.example.stringenumoptions.models.ResponseDto.serializer(), "{}") == alias)
 val dedup = org.example.stringenumdedup.api.models.BetaResponseDto()
 check(dedup.codes?.single() == org.example.stringenumdedup.api.models.AlphaResponseDto.CodesInner.ONE)
 check(Json.decodeFromString(org.example.stringenumdedup.api.models.AlphaResponseDto.serializer(), "{}") == dedup)
 val merged = org.example.stringenummerge.models.ZetaDto()
 check(merged.labels?.get("a") == org.example.stringenummerge.models.AlphaDto.LabelsValue.YES)
 check(org.example.stringenummerge.api.models.AlphaResponseDto() == merged)
 check(Json.decodeFromString(org.example.stringenummerge.models.AlphaDto.serializer(), "{}") == merged)
''' + copy + ' stringEnumCollectionDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runStringEnumCollectionDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.stringenumcollections --api-client-name StringEnumCollections --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.stringenumoptions --api-client-name EnumOptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    for mode in dedup merge; do
      optimization=(--deduplicate-models)
      if [[ "$mode" == merge ]]; then optimization=(--merge-identical-models); fi
      if [[ "$checked" == true ]]; then optimization+=(--validation); fi
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "org.example.stringenum$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runStringEnumCollectionDefaultsContracts
    echo "FRESH STRING ENUM COLLECTION DEFAULTS $style checked=$checked PASS: $project"
  done
done
