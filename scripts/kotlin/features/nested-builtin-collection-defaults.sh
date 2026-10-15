#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/nested-builtin-collection-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedBuiltinCollectionDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/nested-builtin-collection-defaults.json "$project/input.json"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import json
import sys
p = Path(sys.argv[1])
checked = sys.argv[3] == 'true'
d = json.loads((p/'input.json').read_text())
if checked:
    fields = d['components']['schemas']['Holder']['properties']
    fields['required']['minItems'] = 1
    fields['required']['items']['minItems'] = 1
    fields['required']['items']['uniqueItems'] = True
    fields['required']['items']['items']['minimum'] = 0
    fields['rows']['maxItems'] = 3
    fields['rows']['items']['maxItems'] = 3
    fields['listMaps']['items']['maxProperties'] = 3
    fields['listMaps']['items']['additionalProperties']['minLength'] = 1
    fields['mapMaps']['additionalProperties']['maxProperties'] = 2
    fields['mapMaps']['additionalProperties']['additionalProperties']['minimum'] = 1
    patch = d['paths']['/patch']['patch']['requestBody']['content']['application/json']['schema']['properties']
    patch['rows']['minItems'] = 1
    patch['rows']['items']['minItems'] = 1
    patch['rows']['items']['items']['minimum'] = 0
(p/'input.json').write_text(json.dumps(d))
schema = {'properties': {'value': {'additionalProperties': {'additionalProperties': {'type': 'number', 'format': 'float', 'nullable': True}}, 'default': {'b': {'b': None}, 'a': {'b': 1.25, 'a': 1}}}}}
if checked:
    schema['properties']['value']['additionalProperties']['additionalProperties']['minimum'] = 0
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': schema}}}}}} for name in ['alpha', 'beta']}
for mode in ['options', 'dedup', 'merge']:
    document = {'openapi': '3.1.0', 'paths': paths}
    if mode == 'merge':
        document['components'] = {'schemas': {'Alpha': schema, 'Zeta': schema}}
    (p/f'{mode}.json').write_text(json.dumps(document))
# Independent width/depth/encoded-byte compilation probes; no width x depth claim.
wide = {'properties': {f'v{i}': {'type': 'array', 'items': {'type': 'array', 'items': {'type': 'integer'}}, 'default': [[0]]} for i in range(240)}}
leaf = {'type': 'integer'}
if checked:
    leaf['minimum'] = 0
for i in range(16 if checked else 47):
    leaf = {'type': 'array', 'items': leaf}
leaf['default'] = []
large = {'type': 'array', 'items': {'type': 'array', 'items': {'type': 'string'}}, 'default': [['a'*32762]]}
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'Wide': wide, 'Deep': {'properties': {'value': leaf}}, 'Large': {'properties': {'value': large}}}}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.nesteddefaults.models.HolderDto(required = listOf(listOf(2L)))
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
 val optimized = org.example.nestedoptions.models.ResponseDto()
 check(optimized.value == expected)
 val alias = org.example.nestedoptions.api.models.AlphaResponseDto()
 check(Json.decodeFromString(org.example.nestedoptions.models.ResponseDto.serializer(), "{}") == alias)
 val dedup = org.example.nesteddedup.api.models.BetaResponseDto()
 check(dedup.value == expected)
 check(Json.decodeFromString(org.example.nesteddedup.api.models.AlphaResponseDto.serializer(), "{}") == dedup)
 val merged = org.example.nestedmerge.models.ZetaDto()
 check(merged.value == expected && org.example.nestedmerge.api.models.AlphaResponseDto() == merged)
 check(Json.decodeFromString(org.example.nestedmerge.models.AlphaDto.serializer(), "{}") == merged)
 check(org.example.nestedresources.models.WideDto().v239 == listOf(listOf(0)))
 check(org.example.nestedresources.models.DeepDto().value?.isEmpty() == true)
 check(org.example.nestedresources.models.LargeDto().value?.single()?.single()?.length == 32762)
''' + copy + ' nestedBuiltinCollectionDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runNestedBuiltinCollectionDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.nesteddefaults --api-client-name NestedDefaults --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.nestedoptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
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
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "org.example.nested$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNestedBuiltinCollectionDefaultsContracts
    echo "FRESH NESTED BUILTIN COLLECTION DEFAULTS $style checked=$checked PASS: $project"
  done
done
