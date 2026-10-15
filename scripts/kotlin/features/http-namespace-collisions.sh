#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/http-namespaces-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NamespaceCollisionContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    for prefix in org frame input requestBuilder value client basePath json httpClient shadows; do
      "$workspace/openapi-kt" generate --input tests/fixtures/http/namespace-collisions.json --output "$project/build/generated-limits/$prefix" --package-name "$prefix.example.namespaces" --model-name-suffix Dto "${flags[@]}"
      "$workspace/openapi-kt" generate --input tests/fixtures/http/namespace-collisions.json --output "$project/build/generated-limits/$prefix" --package-name "$prefix.example.namespaces" --model-name-suffix Dto "${flags[@]}" --check
    done
    python3 - "$project" <<'PY'
from pathlib import Path
import sys
project = Path(sys.argv[1])
(project / 'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { namespaceCollisionContracts() }
''')
with (project / 'build.gradle.kts').open('a') as output:
    output.write('''
tasks.register<JavaExec>("runNamespaceContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNamespaceContracts
    printf 'FRESH HTTP NAMESPACES PROFILE %s %s PASS\n' "$style" "$checked"
  done
done
printf 'Fresh HTTP namespaces native projects: %s\n' "$workspace"
