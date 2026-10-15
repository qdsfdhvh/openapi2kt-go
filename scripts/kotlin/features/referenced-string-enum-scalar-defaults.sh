#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-string-enum-scalar-defaults-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedStringEnumScalarDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
import json
import sys
from pathlib import Path
p = Path(sys.argv[1]); ref = '#/components/schemas/'
model = {'properties': {'value': {'$ref': ref+'Status', 'default': 'on'}}}
status = {'type': 'string', 'enum': ['on', 'off']}
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': model}}}}}} for name in ['alpha', 'beta']}
(p/'options.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'Status': status}}, 'paths': paths}))
(p/'merge.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'Status': status, 'Alpha': model, 'Zeta': model}}}))
owner = {'properties': {'code': status}}
borrower = {'properties': {'value': {'$ref': ref+'ZetaOwner/properties/code', 'default': 'on'}}}
(p/'owner.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'AlphaOwner': owner, 'ZetaOwner': owner, 'Holder': borrower}}}))
resource_schemas = {
    'Status': status,
    'Wide': {'properties': {f'v{i}': {'$ref': ref+'Status', 'default': 'on'} for i in range(240)}},
    'FullStatus': {'type': 'string', 'enum': [f'v{i}' for i in range(240)]},
    'FullHolder': {'properties': {'value': {'$ref': ref+'FullStatus', 'default': 'v239'}}},
}
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': resource_schemas}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.refstringscalar.models.HolderDto(required = org.example.refstringscalar.models.StatusDto.ON, requiredNullable = null)
 check(value.copy(kotlin = "copy", kotlinx = "copy", serializer = "copy").status === org.example.refstringscalar.models.StatusDto.ON)
 check(value.copy(status = org.example.refstringscalar.models.StatusDto.Unrecognized("future")).status?.serialName == "future")
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val optimized = org.example.refstringscalaroptions.models.ResponseDto()
 check(optimized.value?.serialName == "on")
 check(org.example.refstringscalaroptions.api.models.AlphaResponseDto() == optimized)
 check(org.example.refstringscalaroptions.api.models.BetaResponseDto() == optimized)
 check(Json.decodeFromString(org.example.refstringscalaroptions.models.ResponseDto.serializer(), "{}") == optimized)
 check(org.example.refstringscalardedup.api.models.BetaResponseDto.serializer().descriptor.serialName == "org.example.refstringscalardedup.api.models.AlphaResponseDto")
 check(org.example.refstringscalardedup.api.models.BetaResponseDto().value?.serialName == "on")
 check(org.example.refstringscalarmerge.models.ZetaDto.serializer().descriptor.serialName == "org.example.refstringscalarmerge.models.AlphaDto")
 check(org.example.refstringscalarmerge.models.ZetaDto().value?.serialName == "on")
 val rebound: org.example.refstringscalarowner.models.AlphaOwnerDto.Code? = org.example.refstringscalarowner.models.HolderDto().value
 check(rebound === org.example.refstringscalarowner.models.AlphaOwnerDto.Code.ON)
 check(Json.decodeFromString(org.example.refstringscalarowner.models.HolderDto.serializer(), "{}").value === rebound)
 check(Json.decodeFromString(org.example.refstringscalarowner.models.HolderDto.serializer(), """{"value":"future"}""").value == org.example.refstringscalarowner.models.AlphaOwnerDto.Code.Unrecognized("future"))
 check(org.example.refstringscalarresources.models.WideDto().v239?.serialName == "on")
 check(org.example.refstringscalarresources.models.FullHolderDto().value?.serialName == "v239")
''' + copy + ' referencedStringEnumScalarDefaultsContracts(' + sys.argv[3] + ')\n}\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runReferencedStringEnumScalarDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input tests/fixtures/http/referenced-string-enum-scalar-defaults.json --output "$project/build/generated" --package-name org.example.refstringscalar --api-client-name RefStringScalar --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.refstringscalaroptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    for mode in dedup merge owner resources; do
      optimization=(); input="$project/$mode.json"
      if [[ "$mode" == dedup ]]; then optimization=(--deduplicate-models); input="$project/options.json"; fi
      if [[ "$mode" == merge || "$mode" == owner ]]; then optimization=(--merge-identical-models); fi
      if [[ "$checked" == true ]]; then optimization+=(--validation); fi
      mode_flags=(--input "$input" --output "$project/build/generated-limits/$mode" --package-name "org.example.refstringscalar$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED REFERENCED STRING ENUM SCALAR DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedStringEnumScalarDefaultsContracts
    echo "FRESH REFERENCED STRING ENUM SCALAR DEFAULTS $style checked=$checked PASS: $project"
  done
done
