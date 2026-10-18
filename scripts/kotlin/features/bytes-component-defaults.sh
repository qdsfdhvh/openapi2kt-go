#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/bytes-component-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for validation in off on; do
    project="$workspace/$style-$validation"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BytesComponentDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/bytes-component-defaults.json "$project/input.json"
    flags=()
    [[ "$validation" == off ]] || flags+=(--validation)
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.bytescomponentdefaults --api-client-name BytesComponentDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.bytescomponentdefaults --api-client-name BytesComponentDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}" --check
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else '''
 val value=org.example.bytescomponentdefaults.models.HolderDto()
 check(value.copy(key=byteArrayOf(2)).key!!.contentEquals(byteArrayOf(2)))
 check(value.copy().inline!!.contentEquals(byteArrayOf(1, 2)))
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
'''+copy+''' bytesComponentDefaultsContracts() }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runBytesComponentDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runBytesComponentDefaultsContracts
    echo "FRESH BYTES COMPONENT DEFAULTS PROFILE $style-$validation PASS: $project"
  done
done
echo "Fresh bytes component default native projects: $workspace"
