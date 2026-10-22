#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/nullable-composed-members-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableComposedReferenceMembersContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$checked" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/nullable-composed-reference-members.json').read_text())
def checks(v):
    if isinstance(v,dict):
        if v.get('type')=='string':v['minLength']=2
        for value in list(v.values()):checks(value)
    elif isinstance(v,list):
        for value in v:checks(value)
if checked:checks(d)
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { nullableComposedReferenceMembersContracts('''+str(checked).lower()+''') }
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.composedmembers --model-name-suffix Dto --model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    cat >> "$project/build.gradle.kts" <<'GRADLE'
tasks.register<JavaExec>("runNullableComposedReferenceMembersContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
GRADLE
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED COMPOSED REFERENCE MEMBERS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNullableComposedReferenceMembersContracts
  done
done
echo "Fresh nullable composed member projects: $workspace"
