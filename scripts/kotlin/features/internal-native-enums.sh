#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/internal-native-enums-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  project="$workspace/$style"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/InternalNativeEnumContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  flags=(--validation --model-style "$style")
  "$workspace/openapi-kt" generate --input tests/fixtures/http/internal-native-enums.json --output "$project/build/generated" --package-name org.example.internalnativeenums --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input tests/fixtures/http/internal-native-enums.json --output "$project/build/generated" --package-name org.example.internalnativeenums --model-name-suffix Dto "${flags[@]}" --check
  python3 - "$project" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { internalNativeEnumContracts() }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runInternalNativeEnumContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runInternalNativeEnumContracts
  echo "FRESH INTERNAL NATIVE ENUM PROFILE $style PASS: $project"
done
