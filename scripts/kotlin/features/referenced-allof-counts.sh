#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-allof-counts-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  project="$workspace/$style"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedAllOfCountsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  cp tests/fixtures/http/referenced-allof-counts.json "$project/input.json"
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.referencedcounts --api-client-name ReferencedCounts --model-name-suffix Dto --validation --model-style "$style"
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.referencedcounts --api-client-name ReferencedCounts --model-name-suffix Dto --validation --model-style "$style" --check
  python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else '''
 check(org.example.referencedcounts.models.RecordDto("ok").copy(value="abc").value=="abc")
 check(runCatching { org.example.referencedcounts.models.RecordDto("ok").copy(value="a") }.isFailure)
 val codec=kotlinx.serialization.json.Json
 val decoded=codec.decodeFromString(org.example.referencedcounts.models.RecordDto.serializer(), "{\\"value\\":\\"ok\\",\\"kind\\":\\"other\\"}")
 check(decoded.copy().kind=="record")
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
'''+copy+''' referencedAllOfCountsContracts() }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runReferencedAllOfCountsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedAllOfCountsContracts
  echo "FRESH REFERENCED ALLOF COUNTS PROFILE $style PASS: $project"
done
echo "Fresh referenced allOf counts native projects: $workspace"
