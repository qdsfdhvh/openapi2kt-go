#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/enum-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/EnumDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(Path('tests/fixtures/http/enum-defaults.json').read_text())
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.enumdefaults --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.enumdefaults --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input tests/fixtures/http/enum-default-options.json --output "$project/build/generated-limits" --package-name org.example.enumdefaultoptions --model-name-suffix Dto "${flags[@]}" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names
    "$workspace/openapi-kt" generate --input tests/fixtures/http/enum-default-options.json --output "$project/build/generated-limits" --package-name org.example.enumdefaultoptions --model-name-suffix Dto "${flags[@]}" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import org.example.enumdefaultoptions.models.ResponseDto
import org.example.enumdefaultoptions.api.models.GammaResponseDto
fun main() = runBlocking {
 check(Json.decodeFromString<ResponseDto>("{}").state === ResponseDto.State.READY)
 check(Json.decodeFromString<GammaResponseDto>("{}").state === GammaResponseDto.State.BUSY)
 enumDefaultsContracts('''+str(checked).lower()+''') }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runEnumDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runEnumDefaultsContracts
    echo "FRESH ENUM DEFAULTS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh enum defaults native projects: $workspace"
