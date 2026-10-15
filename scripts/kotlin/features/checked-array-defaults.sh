#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/checked-array-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  project="$workspace/$style"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/CheckedArrayDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  cp tests/fixtures/http/checked-array-defaults.json "$project/input.json"
  flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.checkedarrays --api-client-name CheckedArrays --model-name-suffix Dto --model-style "$style" --validation)
  "$workspace/openapi-kt" generate "${flags[@]}"
  "$workspace/openapi-kt" generate "${flags[@]}" --check
  python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p = Path(sys.argv[1])
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.checkedarrays.models.HolderDto(required = listOf(2))
 check(value.copy(kotlin = "copy").values == value.values)
 val failure = runCatching { value.copy(values = listOf(2, 2)) }.exceptionOrNull()
 check(failure is IllegalArgumentException && failure.message.orEmpty().contains("uniqueItems"))
 val dateFailure = runCatching { value.copy(dates = emptyList()) }.exceptionOrNull()
 check(dateFailure is IllegalArgumentException && dateFailure.message.orEmpty().contains("minItems"))
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
''' + copy + ''' checkedArrayDefaultsContracts() }
''')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runCheckedArrayDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runCheckedArrayDefaultsContracts
  echo "FRESH CHECKED ARRAY DEFAULTS PROFILE $style PASS: $project"
done
