#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/http-servers-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ServerContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    "$workspace/openapi-kt" generate --input tests/fixtures/http/servers.json --output "$project/build/generated" --package-name org.example.servers --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input tests/fixtures/http/servers.json --output "$project/build/generated" --package-name org.example.servers --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PY'
from pathlib import Path
import sys
project = Path(sys.argv[1])
(project / 'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { serverContracts() }
''')
with (project / 'build.gradle.kts').open('a') as output:
    output.write('''
tasks.register<JavaExec>("runServerContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runServerContracts
    printf 'FRESH HTTP SERVERS PROFILE %s %s PASS\n' "$style" "$checked"
  done
done
printf 'Fresh HTTP servers native projects: %s\n' "$workspace"
