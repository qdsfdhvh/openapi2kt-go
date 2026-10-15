#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/primitive-map-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/PrimitiveMapDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    fixture='primitive-map-defaults'
    if [[ "$checked" == true ]]; then fixture='checked-map-defaults'; fi
    cp "tests/fixtures/http/$fixture.json" "$project/input.json"
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.mapdefaults --api-client-name MapDefaults --model-name-suffix Dto --model-style "$style")
    if [[ "$checked" == true ]]; then flags+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p = Path(sys.argv[1])
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.mapdefaults.models.HolderDto(required = mapOf("count" to 2))
 check(value.copy(kotlin = "copy").values == value.values)
 check(value.copy(kotlinx = "copy").dates == value.dates && value.copy().times == value.times)
'''
if sys.argv[2] == 'data_class' and sys.argv[3] == 'true':
    copy += '''
 check(runCatching { value.copy(values = mapOf("a" to 3)) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(dates = emptyMap()) }.exceptionOrNull() is IllegalArgumentException)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
''' + copy + ' primitiveMapDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runPrimitiveMapDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runPrimitiveMapDefaultsContracts
    echo "FRESH MAP DEFAULTS PROFILE $style checked=$checked PASS: $project"
  done
done
