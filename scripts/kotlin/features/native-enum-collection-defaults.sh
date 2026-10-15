#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/native-enum-collection-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NativeEnumCollectionDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/native-enum-collection-defaults.json "$project/input.json"
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
    fields['codes']['items'].update(enum=[None, 1, 2], minimum=2)
    fields['codes']['default'] = [2, None]
    fields['flags']['maxProperties'] = 2
    fields['floats']['additionalProperties'].update(minimum=0.5, maximum=1)
    fields['nested']['properties']['values']['additionalProperties']['enum'] = [1, None, 2]
    patch = d['paths']['/patch']['patch']['requestBody']['content']['application/json']['schema']['properties']
    patch['codes']['minItems'] = 1
    (p/'input.json').write_text(json.dumps(d))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.enumcollections.models.HolderDto(required = listOf(2))
 check(value.copy(kotlin = "copy").flags == value.flags)
'''
if sys.argv[2] == 'data_class' and checked:
    copy += '''
 check(runCatching { value.copy(required = listOf(3)) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(codes = listOf(1)) }.exceptionOrNull() is IllegalArgumentException)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
''' + copy + ' nativeEnumCollectionDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runNativeEnumCollectionDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.enumcollections --api-client-name EnumCollections --model-name-suffix Dto --model-style "$style")
    if [[ "$checked" == true ]]; then flags+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNativeEnumCollectionDefaultsContracts
    echo "FRESH NATIVE ENUM COLLECTION DEFAULTS $style checked=$checked PASS: $project"
  done
done
