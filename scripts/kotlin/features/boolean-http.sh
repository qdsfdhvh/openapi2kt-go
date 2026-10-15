#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/boolean-http-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class;do
 for checked in off on;do
  project="$workspace/$style-$checked";mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BooleanHttpContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  flags=(--model-style "$style");if [[ "$checked" == on ]];then flags+=(--validation);fi
  "$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-http.json --output "$project/build/generated" --package-name org.example.booleanhttp --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-http.json --output "$project/build/generated" --package-name org.example.booleanhttp --model-name-suffix Dto "${flags[@]}" --check
  python3 - "$project" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main()=runBlocking { booleanHttpContracts() }
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''\ntasks.register<JavaExec>("runBooleanHttpContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runBooleanHttpContracts
  echo "FRESH BOOLEAN HTTP PROFILE $style $checked PASS: $project"
 done
done
echo "Fresh Boolean HTTP native projects: $workspace"
