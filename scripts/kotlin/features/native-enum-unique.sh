#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/native-enum-unique-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NativeEnumUniqueContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[3]=='on'
d=json.loads(Path('tests/fixtures/http/native-enum-unique.json').read_text())
def control(v):
    if isinstance(v,dict):
        for key in ('uniqueItems','enum','minimum','multipleOf'):v.pop(key,None)
        for x in v.values():control(x)
    elif isinstance(v,list):
        for x in v:control(x)
if not checked:
    control(d)
    # Checked native/internal alias admission is independent of uniqueness.
    # The unchecked control keeps the same Long wire without those aliases.
    for name in ('Alias','Bridge'):d['components']['schemas'].pop(name)
    def scalar_refs(v):
        if isinstance(v,dict):
            if v.get('$ref') in ('#/components/schemas/Alias','#/components/schemas/Bridge'):
                v['$ref']='#/components/schemas/Code'
            for x in v.values():scalar_refs(x)
        elif isinstance(v,list):
            for x in v:scalar_refs(x)
    scalar_refs(d)
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
copy=''
if sys.argv[2]=='data_class':
    copy='''val original = HolderDto(listOf(true))
    val copied = runCatching { original.copy(flags = listOf(true, true)) }
    check(copied.isFailure == '''+str(checked).lower()+''')'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import org.example.nativeenumunique.models.HolderDto
fun main() = runBlocking {
    '''+copy+'''
    nativeEnumUniqueContracts('''+str(checked).lower()+''')
}
''')
PY
  flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.nativeenumunique --model-name-suffix Dto --model-style "$style")
  if [[ "$checked" == on ]]; then flags+=(--validation); fi
  if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names); fi
  "$workspace/openapi-kt" generate "${flags[@]}"
  "$workspace/openapi-kt" generate "${flags[@]}" --check
  if [[ "$checked" == off ]]; then
    if "$workspace/openapi-kt" validate --input tests/fixtures/http/native-enum-unique.json > "$project/rejected.log" 2>&1; then
      echo 'Validation-off accepted native enum uniqueItems' >&2; exit 1
    fi
    rg -q E_REF_UNSUPPORTED "$project/rejected.log"
  fi
  cat >> "$project/build.gradle.kts" <<'GRADLE'
tasks.register<JavaExec>("runNativeEnumUniqueContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
GRADLE
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
    echo "PREPARED NATIVE ENUM UNIQUE $style $checked: $project"
    continue
  fi
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNativeEnumUniqueContracts
 done
done
echo "Fresh native enum uniqueness projects: $workspace"
