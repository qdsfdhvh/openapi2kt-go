#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-allof-bounds-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  project="$workspace/$style"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedAllOfBoundsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  cp tests/fixtures/http/referenced-allof-bounds.json "$project/input.json"
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.referencedbounds --api-client-name ReferencedBounds --model-name-suffix Dto --validation --model-style "$style"
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.referencedbounds --api-client-name ReferencedBounds --model-name-suffix Dto --validation --model-style "$style" --check
  python3 - "$project" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { referencedAllOfBoundsContracts() }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runReferencedAllOfBoundsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedAllOfBoundsContracts
  echo "FRESH REFERENCED ALLOF BOUNDS PROFILE $style PASS: $project"
done
echo "Fresh referenced allOf bounds native projects: $workspace"
