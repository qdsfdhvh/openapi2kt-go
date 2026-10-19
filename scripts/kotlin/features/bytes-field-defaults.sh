#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/bytes-field-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for validation in off on; do
    project="$workspace/$style-$validation"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/{BytesFieldDefaultsContracts.kt,BytesDefaultTreeContracts.kt} "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/bytes-field-defaults.json "$project/input.json"
    python3 - "$project/input.json" "$validation" <<'PYINPUT'
import json, sys
from pathlib import Path
p = Path(sys.argv[1])
if sys.argv[2] == 'on':
    document = json.loads(p.read_text())
    document['components']['schemas']['Trees']['properties']['rows']['minItems'] = 1
    document['components']['schemas']['Trees']['properties']['rows']['maxItems'] = 3
    p.write_text(json.dumps(document, indent=2)+'\n')
PYINPUT
    flags=()
    [[ "$validation" == off ]] || flags+=(--validation)
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.bytesfielddefaults --api-client-name BytesFieldDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.bytesfielddefaults --api-client-name BytesFieldDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}" --check
    python3 - "$project" "$style" "$validation" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else '''
 val value=org.example.bytesfielddefaults.models.HolderDto(requiredBytes=byteArrayOf(1))
 check(value.copy(requiredBytes=byteArrayOf(2)).requiredBytes.contentEquals(byteArrayOf(2)))
 check(value.copy().optionalBytes!!.contentEquals(byteArrayOf(4)))
 val tree=org.example.bytesfielddefaults.models.TreesDto(requiredRows=listOf(byteArrayOf(9)))
 check(tree.copy().rows === tree.rows)
 check(tree.copy(requiredRows=listOf(byteArrayOf(2))).requiredRows.single().contentEquals(byteArrayOf(2)))
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
'''+copy+''' bytesFieldDefaultsContracts('''+str(sys.argv[3]=='on').lower()+''') }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runBytesFieldDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runBytesFieldDefaultsContracts
    echo "FRESH BYTES FIELD DEFAULTS PROFILE $style-$validation PASS: $project"
  done
done
echo "Fresh bytes field default native projects: $workspace"
