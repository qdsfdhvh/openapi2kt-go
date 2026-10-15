#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/native-enum-bounds-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  project="$workspace/$style"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NativeEnumBoundsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  flags=(--validation --model-style "$style")
  "$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-bounds.json --output "$project/build/generated" --package-name org.example.nativeenumbounds --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input tests/fixtures/http/native-enum-bounds.json --output "$project/build/generated" --package-name org.example.nativeenumbounds --model-name-suffix Dto "${flags[@]}" --check
  python3 - "$project" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { nativeEnumBoundsContracts() }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runNativeEnumBoundsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNativeEnumBoundsContracts
  echo "FRESH NATIVE ENUM BOUNDS PROFILE $style PASS: $project"
done
