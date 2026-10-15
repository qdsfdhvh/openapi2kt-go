#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/merge-exclusions-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  project="$workspace/$style"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/MergeExclusionContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  flags=(--merge-identical-models --validation --model-style "$style" --hoist-shared-shapes --deduplicate-models --lock-hoisted-names --qualify-hoisted-names --runtime-package org.example.exclusions.runtime --exclude-merge-model-pattern 'org\.example\.exclusions\.models\.ZetaDto' --exclude-merge-model-suffix DtoDto)
  "$workspace/openapi-kt" generate --input tests/fixtures/http/merge-exclusions.json --output "$project/build/generated" --package-name org.example.exclusions --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input tests/fixtures/http/merge-exclusions.json --output "$project/build/generated" --package-name org.example.exclusions --model-name-suffix Dto "${flags[@]}" --check
  python3 - "$project" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1])
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { mergeExclusionContracts() }
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''\ntasks.register<JavaExec>("runMergeExclusionContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runMergeExclusionContracts
  printf 'FRESH MERGE EXCLUSIONS PROFILE %s PASS: %s\n' "$style" "$project"
done
printf 'Fresh native Go CLI merge exclusion projects: %s\n' "$workspace"
