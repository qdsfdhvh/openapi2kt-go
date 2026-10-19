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
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableComponentDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/nullable-component-defaults.json "$project/input.json"
    flags=()
    [[ "$validation" == off ]] || flags+=(--validation)
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.nulldefaults --api-client-name NullableDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.nulldefaults --api-client-name NullableDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}" --check
    cat > "$project/src/main/kotlin/org/example/contracts/Main.kt" <<'KOTLIN'
package org.example.contracts

import kotlinx.coroutines.runBlocking

fun main() = runBlocking { nullableComponentDefaultsContracts() }
KOTLIN
    python3 - "$project" <<'PYMAIN'
from pathlib import Path
import sys
p = Path(sys.argv[1])
b = p / 'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runNullableComponentDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNullableComponentDefaultsContracts
    echo "FRESH NULLABLE COMPONENT DEFAULTS PROFILE $style-$validation PASS: $project"
  done
done
echo "Fresh nullable component default native projects: $workspace"
