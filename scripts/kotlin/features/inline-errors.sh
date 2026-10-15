#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/inline-errors-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/InlineErrorContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(Path('tests/fixtures/http/inline-errors.json').read_text())
if sys.argv[2]=='on':
    def bounds(v):
        if isinstance(v,dict):
            if v.get('type')=='string':v['minLength']=1
            for child in list(v.values()):bounds(child)
        elif isinstance(v,list):
            for child in v:bounds(child)
    for path in d['paths'].values():
        for operation in path.values():
            for response in operation.get('responses', {}).values(): bounds(response)
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.inlineerrors --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.inlineerrors --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input tests/fixtures/http/inline-error-options.json --output "$project/build/generated-limits" --package-name org.example.inlineerroroptions --model-name-suffix Dto "${flags[@]}" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names
    "$workspace/openapi-kt" generate --input tests/fixtures/http/inline-error-options.json --output "$project/build/generated-limits" --package-name org.example.inlineerroroptions --model-name-suffix Dto "${flags[@]}" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import org.example.inlineerroroptions.models.Error400Dto
import org.example.inlineerroroptions.api.models.AlphaError400Dto
import org.example.inlineerroroptions.api.models.BetaError400Dto
fun main() = runBlocking {
 check(Json.decodeFromString<AlphaError400Dto>("""{"message":"given"}""").message == "given")
 check(Json.decodeFromString<BetaError400Dto>("""{"message":"given"}""") is Error400Dto)
 inlineErrorContracts('''+str(checked).lower()+''') }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runInlineErrorContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runInlineErrorContracts
    echo "FRESH INLINE ERRORS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh model defaults native projects: $workspace"
