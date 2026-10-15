#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/boolean-root-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class;do
 for checked in off on;do
  project="$workspace/$style-$checked";mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BooleanRootContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  flags=(--model-style "$style");if [[ "$checked" == on ]];then flags+=(--validation);fi
  "$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-roots.json --output "$project/build/generated" --package-name org.example.booleanroots --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input tests/fixtures/http/boolean-roots.json --output "$project/build/generated" --package-name org.example.booleanroots --model-name-suffix Dto "${flags[@]}" --check
  python3 - "$project" "$style" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);extra='' if sys.argv[2]=='plain_class' else '''val value=org.example.booleanroots.models.HolderDto(value=kotlinx.serialization.json.JsonNull,chain=kotlinx.serialization.json.JsonNull,rows=emptyList(),mapping=emptyMap(),nullable=null)
    check(value.copy(value=kotlinx.serialization.json.JsonPrimitive("copy")).value==kotlinx.serialization.json.JsonPrimitive("copy"))'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main()=runBlocking {
    '''+extra+'''
    booleanRootContracts()
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''\ntasks.register<JavaExec>("runBooleanRootContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runBooleanRootContracts
  echo "FRESH BOOLEAN ROOT PROFILE $style $checked PASS: $project"
 done
done
echo "Fresh Boolean root native projects: $workspace"
