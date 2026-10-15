#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-string-enum-null-member-defaults-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedStringEnumNullMemberDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
import json
import sys
from pathlib import Path
p = Path(sys.argv[1]); ref = '#/components/schemas/'
doc = json.loads(Path('tests/fixtures/http/referenced-string-enum-null-member-defaults.json').read_text())
if sys.argv[3] == 'true':
    doc['components']['schemas']['Rows']['maxItems'] = 3
    doc['components']['schemas']['Maps']['maxProperties'] = 2
    doc['components']['schemas']['Nested']['maxItems'] = 2
(p/'input.json').write_text(json.dumps(doc))
status = {'type': ['string', 'null'], 'enum': [None, 'on', 'off']}
model = {'properties': {'scalar': {'$ref': ref+'Status', 'default': 'on'}, 'rows': {'$ref': ref+'Rows', 'default': [None, 'on']}}}
rows = {'type': 'array', 'items': {'$ref': ref+'Status'}}
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': model}}}}}} for name in ['alpha', 'beta']}
(p/'options.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'Status': status, 'Rows': rows}}, 'paths': paths}))
(p/'merge.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'Status': status, 'Rows': rows, 'Alpha': model, 'Zeta': model}}}))
owner = {'properties': {'code': status}}
borrower = {'properties': {'value': {'$ref': ref+'ZetaOwner/properties/code', 'default': 'on'}}}
(p/'owner.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'AlphaOwner': owner, 'ZetaOwner': owner, 'Holder': borrower}}}))
resources = {
    'Status': status,
    'Wide': {'properties': {f'v{i}': {'$ref': ref+'Status', 'default': 'on'} for i in range(240)}},
    'FullStatus': {'type': ['string', 'null'], 'enum': [None] + [f'v{i}' for i in range(239)]},
    'FullHolder': {'properties': {'value': {'$ref': ref+'FullStatus', 'default': 'v238'}}},
}
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': resources}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.refnullstrings.models.HolderDto(required = null)
 check(value.copy(kotlin = "copy", kotlinx = "copy", serializer = "copy").rows == value.rows)
 check(value.copy(scalar = org.example.refnullstrings.models.StatusDto.Unrecognized("future")).scalar?.serialName == "future")
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val optimized = org.example.refnullstringsoptions.models.ResponseDto()
 check(optimized.scalar?.serialName == "on" && optimized.rows?.first() == null)
 check(org.example.refnullstringsoptions.api.models.AlphaResponseDto() == optimized)
 check(org.example.refnullstringsoptions.api.models.BetaResponseDto() == optimized)
 check(Json.decodeFromString(org.example.refnullstringsoptions.models.ResponseDto.serializer(), "{}") == optimized)
 check(org.example.refnullstringsdedup.api.models.BetaResponseDto.serializer().descriptor.serialName == "org.example.refnullstringsdedup.api.models.AlphaResponseDto")
 check(org.example.refnullstringsdedup.api.models.BetaResponseDto().rows?.first() == null)
 check(org.example.refnullstringsmerge.models.ZetaDto.serializer().descriptor.serialName == "org.example.refnullstringsmerge.models.AlphaDto")
 check(org.example.refnullstringsmerge.models.ZetaDto().scalar?.serialName == "on")
 val rebound: org.example.refnullstringsowner.models.AlphaOwnerDto.Code? = org.example.refnullstringsowner.models.HolderDto().value
 check(rebound === org.example.refnullstringsowner.models.AlphaOwnerDto.Code.ON)
 check(Json.decodeFromString(org.example.refnullstringsowner.models.HolderDto.serializer(), "{}").value === rebound)
 check(org.example.refnullstringsresources.models.WideDto().v239?.serialName == "on")
 check(org.example.refnullstringsresources.models.FullHolderDto().value?.serialName == "v238")
 check(org.example.refnullstringsresources.models.FullStatusDto.entries.size == 239)
''' + copy + ' referencedStringEnumNullMemberDefaultsContracts(' + sys.argv[3] + ')\n}\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runReferencedStringEnumNullMemberDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.refnullstrings --api-client-name RefNullStrings --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.refnullstringsoptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
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
      mode_flags=(--input "$input" --output "$project/build/generated-limits/$mode" --package-name "org.example.refnullstrings$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED REFERENCED STRING ENUM NULL MEMBER DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedStringEnumNullMemberDefaultsContracts
    echo "FRESH REFERENCED STRING ENUM NULL MEMBER DEFAULTS $style checked=$checked PASS: $project"
  done
done
