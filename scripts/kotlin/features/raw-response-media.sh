#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/raw-response-media-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RawResponseMediaContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PY'
from pathlib import Path
import json, sys
project = Path(sys.argv[1]); checked = sys.argv[2] == 'on'
document = json.loads(Path('tests/fixtures/http/raw-response-media.json').read_text())
if checked: document['components']['schemas']['Text']['minLength'] = 1
(project/'input.json').write_text(json.dumps(document, indent=2)+'\n')
(project/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { rawResponseMediaContracts('''+str(checked).lower()+''') }
''')
build = project/'build.gradle.kts'
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runRawResponseMediaContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.rawmedia --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.rawmedia --model-name-suffix Dto "${flags[@]}" --check
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runRawResponseMediaContracts
    echo "FRESH RAW RESPONSE MEDIA PROFILE $style $checked PASS: $project"
  done
done
