#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/primitive-tuple-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/PrimitiveTupleContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  flags=(--model-style "$style");if [[ "$checked" == on ]];then flags+=(--validation);fi
  "$workspace/openapi-kt" generate --input tests/fixtures/http/primitive-tuples.json --output "$project/build/generated" --package-name org.example.tuples --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input tests/fixtures/http/primitive-tuples.json --output "$project/build/generated" --package-name org.example.tuples --model-name-suffix Dto "${flags[@]}" --check
  python3 - "$project" "$style" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);plain=sys.argv[2]=='plain_class'
extra='' if plain else '''val source=org.example.tuples.models.HolderDto(listOf("v"),listOf(1L),emptyList(),listOf("v"),null,emptyMap())
    check(source.copy(strings=listOf("copy")).strings==listOf("copy"))'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
    '''+extra+'''
    primitiveTupleContracts()
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''\ntasks.register<JavaExec>("runPrimitiveTupleContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runPrimitiveTupleContracts
  printf 'FRESH PRIMITIVE TUPLE PROFILE %s %s PASS: %s\n' "$style" "$checked" "$project"
 done
done
printf 'Fresh native Go CLI tuple projects: %s\n' "$workspace"
