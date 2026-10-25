#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/inline-error-roots-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/InlineErrorRootsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(Path('tests/fixtures/http/inline-error-roots.json').read_text())
if sys.argv[2]=='on':
    def bounds(v):
        if isinstance(v,dict):
            for field in v.get('properties',{}).values():
                if field.get('type')=='string':field['minLength']=1
            for child in list(v.values()):bounds(child)
        elif isinstance(v,list):
            for child in v:bounds(child)
    for path in d['paths'].values():
        for operation in path.values():
            for response in operation.get('responses', {}).values(): bounds(response)
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
wide={'openapi':'3.1.0','paths':{'/fetch':{'get':{'operationId':'fetch','responses':{'204':{},'400':{'content':{'application/json':{'schema':{'allOf':[{'properties':{'p'+str(i):{'type':'string'} for i in range(240)}}]}}}}}}}}}
(p/'wide.json').write_text(json.dumps(wide)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.errorroots --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.errorroots --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.errorrootsoptimized --model-name-suffix Dto "${flags[@]}" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.errorrootsoptimized --model-name-suffix Dto "${flags[@]}" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names --check
    "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.errorrootswide --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.errorrootswide --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import org.example.errorrootsoptimized.api.models.FetchError400Dto
fun main() = runBlocking {
 check(org.example.errorrootswide.api.models.FetchError400Dto().p239 == null)
 check(Json.decodeFromString<FetchError400Dto>("""{"message":"given"}""").message == "given")
 inlineErrorRootsContracts('''+str(checked).lower()+''') }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runInlineErrorRootsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runInlineErrorRootsContracts
    echo "FRESH INLINE ERROR ROOTS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh model defaults native projects: $workspace"
