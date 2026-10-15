#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-temporal-scalar-field-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedTemporalScalarFieldDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import json, sys
p = Path(sys.argv[1]); checked = sys.argv[3] == 'true'; ref = '#/components/schemas/'
d = json.loads(Path('tests/fixtures/http/referenced-temporal-scalar-field-defaults.json').read_text())
(p/'input.json').write_text(json.dumps(d))
model = {'properties': {'value': {'$ref': ref+'Stamp', 'default': '2024-02-29T12:34:56+02:30'}}}
paths = {f'/{n}': {'get': {'operationId': n, 'responses': {'200': {'content': {'application/json': {'schema': model}}}}}} for n in ['alpha', 'beta']}
for mode in ['options', 'dedup', 'merge']:
    schemas = {'Stamp': {'type': 'string', 'format': 'date-time', 'nullable': True}}
    if mode == 'merge': schemas.update(Alpha=model, Zeta=model)
    (p/f'{mode}.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': schemas}, 'paths': paths}))
# Independent probes: scalar source48/slots240, existing builtin checked16 and
# default32KiB beside a temporal default. No temporal constraints or combined claim.
schemas = {'Stamp': {'type': 'string', 'format': 'date-time', 'nullable': True}, 'Hops': {'properties': {}}}
schemas['Wide'] = {'properties': {f'v{i}': {'$ref': ref+'Stamp', 'default': '2024-02-29T00:00:00Z'} for i in range(240)}}
for i in range(46):
    schemas['Hops']['properties'][f'h{i}'] = {'$ref': ref+('Stamp' if i == 0 else f'Hops/properties/h{i-1}'), 'nullable': False}
schemas['Source48'] = {'properties': {'value': {'$ref': ref+'Hops/properties/h45', 'default': '2024-02-29T00:00:00Z'}}}
schemas['Text'] = {'type': 'string'}
schemas['Large'] = {'properties': {'value': {'$ref': ref+'Text', 'default': 'a'*32766}, 'stamp': {'$ref': ref+'Stamp', 'default': '2024-02-29T00:00:00Z'}}}
leaf = {'type': 'integer', **({'minimum': 0} if checked else {})}
for _ in range(16): leaf = {'type': 'array', 'items': leaf}
schemas['Deep'] = {'properties': {'value': leaf, 'own': {'$ref': ref+'Stamp', 'default': '2024-02-29T00:00:00Z'}}}
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': schemas}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.reftemporal.models.HolderDto(required = LocalDate.parse("2024-02-29"))
 check(value.copy(kotlin = "copy", kotlinx = "copy").stamp == value.stamp)
 check(value.copy(day = null).day == null)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlin.time.Instant
import kotlinx.coroutines.runBlocking
import kotlinx.datetime.LocalDate
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val expected = Instant.parse("2024-02-29T10:04:56Z")
 val optimized = org.example.reftemporaloptions.models.ResponseDto()
 val actual: org.example.reftemporaloptions.models.StampDto = optimized.value
 val builtin: Instant? = actual
 check(builtin == expected)
 check(org.example.reftemporaloptions.api.models.BetaResponseDto() == optimized)
 check(org.example.reftemporaloptions.models.ResponseDto.serializer().descriptor.serialName == "org.example.reftemporaloptions.models.ResponseDto")
 val dedup = org.example.reftemporaldedup.api.models.BetaResponseDto()
 check(dedup.value == expected && dedup::class == org.example.reftemporaldedup.api.models.AlphaResponseDto()::class)
 check(org.example.reftemporaldedup.api.models.BetaResponseDto.serializer().descriptor.serialName == "org.example.reftemporaldedup.api.models.AlphaResponseDto")
 val merged = org.example.reftemporalmerge.models.ZetaDto()
 check(merged.value == expected && org.example.reftemporalmerge.api.models.AlphaResponseDto() == merged)
 check(org.example.reftemporalmerge.models.ZetaDto.serializer().descriptor.serialName == "org.example.reftemporalmerge.models.AlphaDto")
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
   val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
   check(codec.decodeFromString(org.example.reftemporaloptions.models.ResponseDto.serializer(), "{}") == optimized)
   check(codec.decodeFromString(org.example.reftemporaldedup.api.models.AlphaResponseDto.serializer(), "{}") == dedup)
   check(codec.decodeFromString(org.example.reftemporalmerge.models.AlphaDto.serializer(), "{}") == merged)
 }
 check(org.example.reftemporalresources.models.WideDto().v239 == Instant.parse("2024-02-29T00:00:00Z"))
 check(org.example.reftemporalresources.models.Source48Dto().value == Instant.parse("2024-02-29T00:00:00Z"))
 check(org.example.reftemporalresources.models.LargeDto().value?.length == 32766)
 check(org.example.reftemporalresources.models.DeepDto().own == Instant.parse("2024-02-29T00:00:00Z"))
''' + copy + ' referencedTemporalScalarFieldDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'; b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runReferencedTemporalScalarFieldDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.reftemporal --api-client-name ReferencedTemporal --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.reftemporaloptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
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
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "org.example.reftemporal$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED REFERENCED TEMPORAL SCALAR FIELD DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedTemporalScalarFieldDefaultsContracts
    echo "FRESH REFERENCED TEMPORAL SCALAR FIELD DEFAULTS $style checked=$checked PASS: $project"
  done
done
