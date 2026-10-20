#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/nullable-component-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for validation in off on; do
    project="$workspace/$style-$validation"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/{NullableComponentDefaultsContracts,ComponentNullDefaultsContracts}.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/nullable-component-defaults.json "$project/input.json"
    python3 - "$project" "$validation" "$style" <<'PY'
from pathlib import Path
import json, sys
p = Path(sys.argv[1]); checked = sys.argv[2] == 'on'
schemas = {'LargeMode': {'type': ['string', 'null'], 'enum': [f'v{i}' for i in range(240)], 'default': None}}
if checked:
    for name, kind, fmt, values in [
        ('Int', 'integer', 'int32', [1,2]),
        ('Long', 'integer', 'int64', [-9223372036854775808,9223372036854775807]),
        ('Float', 'number', 'float', [-1.5,2.5]),
        ('Double', 'number', 'double', [1.25,2.75]),
        ('Flag', 'boolean', None, [True,False]),
        ('BigCodes', 'integer', 'int32', list(range(240))),
    ]:
        schemas[name] = {'type': kind, 'nullable': True, 'enum': values, 'default': None}
        if fmt: schemas[name]['format'] = fmt
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': schemas}}))
body = '''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.serializer
fun main() = runBlocking {
 nullableComponentDefaultsContracts()
 check(org.example.nullfactoryresources.models.LargeModeDto.entries.size == 240)
'''
if sys.argv[3] == 'data_class':
    body += ''' val value = org.example.nulldefaults.models.HolderDto()
 check(value.copy(nilName = "copy").nilName == "copy")
 check(value.copy(nilName = null).nilName == null && value.copy().nilMode == null)
'''
body += ' for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {\n val codec = Json { encodeDefaults=defaults; explicitNulls=nulls }\n'
for name in schemas:
    # Only the physical String-enum classifier needs an explicit nullable view.
    ty = f'org.example.nullfactoryresources.models.{name}Dto' + ('?' if name == 'LargeMode' else '')
    factory = f'org.example.nullfactoryresources.models.`{name}Dto default`()'
    body += f' val value{name}: {ty} = {factory}\n'
    body += f' check(value{name} == null && serializer<{ty}>().descriptor.isNullable)\n'
    body += f' check(codec.encodeToJsonElement(serializer<{ty}>(), value{name}) === JsonNull)\n'
    body += f' check(codec.decodeFromJsonElement(serializer<{ty}>(), JsonNull) == null)\n'
body += ' }\n}\n'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(body)
b = p/'build.gradle.kts'
b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runNullableComponentDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=()
    [[ "$validation" == off ]] || flags+=(--validation)
    main_flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.nulldefaults --api-client-name NullableDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}")
    "$workspace/openapi-kt" generate "${main_flags[@]}"
    "$workspace/openapi-kt" generate "${main_flags[@]}" --check
    resources=(--input "$project/resources.json" --output "$project/build/generated-limits" --package-name org.example.nullfactoryresources --model-name-suffix Dto --model-style "$style" "${flags[@]}")
    "$workspace/openapi-kt" generate "${resources[@]}"
    "$workspace/openapi-kt" generate "${resources[@]}" --check
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED NULLABLE COMPONENT DEFAULTS $style-$validation: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNullableComponentDefaultsContracts
    echo "FRESH NULLABLE COMPONENT DEFAULTS PROFILE $style-$validation PASS: $project"
  done
done
echo "Fresh nullable component default native projects: $workspace"
