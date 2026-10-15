#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-scalar-field-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedScalarFieldDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import json, sys
p = Path(sys.argv[1]); checked = sys.argv[3] == 'true'; ref = '#/components/schemas/'
d = json.loads(Path('tests/fixtures/http/referenced-scalar-field-defaults.json').read_text())
s = d['components']['schemas']
if checked:
    s['Count'].update(maximum=10, multipleOf=2)
    s['Label'].update(minLength=1, maxLength=32)
    s['Ratio'].update(minimum=1, maximum=2)
    s['DoubleValue'].update(exclusiveMinimum=1)
(p/'input.json').write_text(json.dumps(d))
model = {'properties': {'value': {'$ref': ref+'Value', 'default': 1.000000001}}}
paths = {f'/{n}': {'get': {'operationId': n, 'responses': {'200': {'content': {'application/json': {'schema': model}}}}}} for n in ['alpha', 'beta']}
for mode in ['options', 'dedup', 'merge']:
    schemas = {'Value': {'type': 'number', 'format': 'float', 'nullable': True, **({'minimum': 0} if checked else {})}}
    if mode == 'merge': schemas.update(Alpha=model, Zeta=model)
    (p/f'{mode}.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': schemas}, 'paths': paths}))
# Independent resource probes. No cap or heap increase and no combined guarantee.
schemas = {'Value': {'type': 'integer', 'format': 'int64', 'nullable': True}, 'Hops': {'properties': {}}}
if checked: schemas['Value']['minimum'] = 0
schemas['Wide'] = {'properties': {f'v{i}': {'$ref': ref+'Value', 'default': 2} for i in range(240)}}
for i in range(46):
    schemas['Hops']['properties'][f'h{i}'] = {'$ref': ref+('Source' if i == 0 else f'Hops/properties/h{i-1}'), 'nullable': False}
schemas['Source'] = {'type': 'string', 'nullable': True}
schemas['Source48'] = {'properties': {'value': {'$ref': ref+'Hops/properties/h45', 'default': 'source'}}}
schemas['Large'] = {'properties': {'value': {'$ref': ref+'Source', 'default': 'a'*32766}}}
leaf = {'type': 'integer', **({'minimum': 0} if checked else {})}
for _ in range(16): leaf = {'type': 'array', 'items': leaf}
schemas['Deep'] = {'properties': {'value': leaf, 'own': {'$ref': ref+'Value', 'default': 2}}}
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': schemas}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.refscalars.models.HolderDto(required = 2L)
 check(value.copy(kotlin = "copy").count == 2L)
'''
if sys.argv[2] == 'data_class' and checked:
    copy += '''
 check(runCatching { value.copy(required = 12L) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(ratio = 0.0f) }.exceptionOrNull() is IllegalArgumentException)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val optimized = org.example.refscalaroptions.models.ResponseDto()
 val actual: org.example.refscalaroptions.models.ValueDto = optimized.value
 val builtin: Float? = actual
 check(builtin == 1.0f)
 check(org.example.refscalaroptions.api.models.BetaResponseDto() == optimized)
 check(org.example.refscalaroptions.models.ResponseDto.serializer().descriptor.serialName == "org.example.refscalaroptions.models.ResponseDto")
 val dedup = org.example.refscalardedup.api.models.BetaResponseDto()
 check(Json.decodeFromString(org.example.refscalardedup.api.models.AlphaResponseDto.serializer(), "{}") == dedup)
 val merged = org.example.refscalarmerge.models.ZetaDto()
 check(merged.value == 1.0f && org.example.refscalarmerge.api.models.AlphaResponseDto() == merged)
 check(org.example.refscalarmerge.models.ZetaDto.serializer().descriptor.serialName == "org.example.refscalarmerge.models.AlphaDto")
 check(org.example.refscalarresources.models.WideDto().v239 == 2L)
 check(org.example.refscalarresources.models.Source48Dto().value == "source")
 check(org.example.refscalarresources.models.LargeDto().value?.length == 32766)
 check(org.example.refscalarresources.models.DeepDto().own == 2L)
''' + copy + ' referencedScalarFieldDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'; b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runReferencedScalarFieldDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.refscalars --api-client-name ReferencedScalars --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.refscalaroptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
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
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "org.example.refscalar$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED REFERENCED SCALAR FIELD DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedScalarFieldDefaultsContracts
    echo "FRESH REFERENCED SCALAR FIELD DEFAULTS $style checked=$checked PASS: $project"
  done
done
