#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/primitive-array-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for validation in off on; do
    project="$workspace/$style-$validation"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/PrimitiveArrayDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/primitive-array-defaults.json "$project/input.json"
    flags=()
    [[ "$validation" == off ]] || flags+=(--validation)
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.arraydefaults --api-client-name ArrayDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.arraydefaults --api-client-name ArrayDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}" --check
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else '''
 val value=org.example.arraydefaults.models.HolderDto(required=listOf("provided"))
 check(value.copy(kotlin="copy").longs==value.longs)
 check(value.copy(kotlin="copy").dates==value.dates && value.copy().times==value.times)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
'''+copy+''' primitiveArrayDefaultsContracts() }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runPrimitiveArrayDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runPrimitiveArrayDefaultsContracts
    echo "FRESH PRIMITIVE ARRAY DEFAULTS PROFILE $style-$validation PASS: $project"
  done
done
echo "Fresh primitive array default native projects: $workspace"
