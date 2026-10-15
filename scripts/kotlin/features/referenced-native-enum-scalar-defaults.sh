#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-native-enum-scalar-defaults-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedNativeEnumScalarDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
import json
import sys
from pathlib import Path
p = Path(sys.argv[1]); checked = sys.argv[3] == 'true'; ref = '#/components/schemas/'
doc = json.loads(Path('tests/fixtures/http/referenced-native-enum-scalar-defaults.json').read_text())
extra = ''
if checked:
    schemas = doc['components']['schemas']
    schemas['Code'].update(minimum=1, maximum=2)
    schemas['FloatCode']['exclusiveMinimum'] = 0
    schemas['Source'] = {'properties': {'code': {'type': ['integer', 'null'], 'enum': [None, 1, 2], 'minimum': 1}}}
    schemas['Bridge'] = {'$ref': ref+'Source/properties/code'}
    schemas['Alias'] = {'$ref': ref+'Bridge'}
    schemas['CheckedHolder'] = {'properties': {'value': {'$ref': ref+'Alias', 'nullable': False, 'default': 2}, 'nil': {'$ref': ref+'Alias', 'nullable': True, 'default': None}}}
    extra = '''
 val internal = org.example.refnativescalar.models.CheckedHolderDto()
 check(internal.value == 2 && internal.nil == null)
 check(Json.decodeFromString(org.example.refnativescalar.models.CheckedHolderDto.serializer(), "{}") == internal)
 check(runCatching { org.example.refnativescalar.models.CheckedHolderDto(value = 9) }.exceptionOrNull() is IllegalArgumentException)
'''
(p/'input.json').write_text(json.dumps(doc))
code = {'type': 'number', 'format': 'float', 'enum': [1, 1.000000001, 2]}
model = {'properties': {'value': {'$ref': ref+'Code', 'default': 1.000000001}}}
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': model}}}}}} for name in ['alpha', 'beta']}
(p/'options.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'Code': code}}, 'paths': paths}))
wide = {'properties': {f'v{i}': {'$ref': ref+'Code', 'default': 2} for i in range(240)}}
resource_schemas = {
    'Code': {'type': 'integer', 'enum': [1, 2]},
    'Wide': wide,
    'FullCode': {'type': 'integer', 'enum': list(range(1, 241))},
    'FullHolder': {'properties': {'value': {'$ref': ref+'FullCode', 'default': 240}}},
}
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': resource_schemas}}))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.refnativescalar.models.HolderDto(required = 1, requiredNullable = null)
 check(value.copy(kotlin = "copy", companion = "copy", serializer = "copy").minLong == Long.MIN_VALUE)
 check(value.copy(i = 1, f = 1.0f).i == 1)
'''
if checked and sys.argv[2] == 'data_class':
    copy += ' check(runCatching { value.copy(i = 9) }.exceptionOrNull() is IllegalArgumentException)\n'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 check(org.example.refnativescalaroptions.models.ResponseDto().value == 1.0f)
 check(org.example.refnativescalaroptions.api.models.AlphaResponseDto().value == 1.0f)
 check(org.example.refnativescalaroptions.api.models.BetaResponseDto().value == 1.0f)
 check(org.example.refnativescalarresources.models.WideDto().v239 == 2)
 check(org.example.refnativescalarresources.models.FullHolderDto().value == 240)
''' + copy + extra + ' referencedNativeEnumScalarDefaultsContracts(' + sys.argv[3] + ')\n}\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runReferencedNativeEnumScalarDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.refnativescalar --api-client-name RefNativeScalar --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.refnativescalaroptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    resources=(--input "$project/resources.json" --output "$project/build/generated-limits" --package-name org.example.refnativescalarresources --model-name-suffix Dto --model-style "$style")
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); resources+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    "$workspace/openapi-kt" generate "${resources[@]}"
    "$workspace/openapi-kt" generate "${resources[@]}" --check
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED REFERENCED NATIVE ENUM SCALAR DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedNativeEnumScalarDefaultsContracts
    echo "FRESH REFERENCED NATIVE ENUM SCALAR DEFAULTS $style checked=$checked PASS: $project"
  done
done
