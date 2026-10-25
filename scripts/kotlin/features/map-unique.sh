#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/map-unique-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/MapUniqueContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[3]=='on'
d=json.loads(Path('tests/fixtures/http/map-unique.json').read_text())
def control(v):
    if isinstance(v,dict):
        v.pop('uniqueItems',None)
        v.pop('minProperties',None)
        v.pop('minLength',None)
        for x in v.values():control(x)
    elif isinstance(v,list):
        for x in v:control(x)
if not checked:control(d)
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
wide=''
if checked:
    spec={'openapi':'3.1.0','components':{'schemas':{'Wide':{'properties':{
        'p'+str(i):{'type':'array','uniqueItems':True,'items':{'type':'object','additionalProperties':{'type':'string', 'format':['uuid','date','date-time'][i%3]}}} for i in range(240)
    }}}}}
    spec['components']['schemas']['StructuralWide']={'properties':{
        'p'+str(i):{'type':'array','uniqueItems':True,'items':{'type':'array','items':{'type':'number','format':'float'}}} for i in range(240)
    }}
    deep={'type':'string'}
    for _ in range(15):deep={'type':'array','items':deep}
    spec['components']['schemas']['StructuralDeep']={'properties':{}}
    # The outer checked array plus 15 child edges reaches the existing limit.
    spec['components']['schemas']['StructuralDeep']['properties']['value']={'type':'array','uniqueItems':True,'items':deep}
    (p/'wide.json').write_text(json.dumps(spec)+'\n')
    wide='''org.example.mapwide.models.WideDto()
    org.example.mapwide.models.StructuralDeepDto(emptyList())
    org.example.mapwide.models.StructuralWideDto(p0 = listOf(listOf(1f), listOf(2f)))
    check(runCatching { org.example.mapwide.models.StructuralWideDto(p239 = listOf(listOf(0f), listOf(-0f))) }.isFailure)'''
copy=''
if sys.argv[2]=='data_class':
    copy='''val id = mapOf("a" to "one")
    val original = HolderDto(listOf(id))
    val copied = runCatching { original.copy(directValues = listOf(id, id)) }
    check(copied.isFailure == '''+str(checked).lower()+''')
    check(runCatching { original.copy(listRows = listOf(listOf(1), listOf(1))) }.isFailure == '''+str(checked).lower()+''')
    val instant = kotlin.time.Instant.parse("2026-01-02T03:04:05Z")
    check(runCatching { original.copy(instants = listOf(mapOf("a" to instant), mapOf("a" to instant))) }.isFailure == '''+str(checked).lower()+''')'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import org.example.mapunique.models.HolderDto
fun main() = runBlocking {
    '''+copy+'''
    '''+wide+'''
    mapUniqueContracts('''+str(checked).lower()+''')
}
''')
PY
  flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.mapunique --model-name-suffix Dto --model-style "$style")
  if [[ "$checked" == on ]]; then flags+=(--validation); fi
  if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names); fi
  "$workspace/openapi-kt" generate "${flags[@]}"
  "$workspace/openapi-kt" generate "${flags[@]}" --check
  if [[ "$checked" == on ]]; then
    "$workspace/openapi-kt" generate --validation --input "$project/wide.json" --output "$project/build/generated-limits/map" --package-name org.example.mapwide --model-name-suffix Dto --model-style "$style"
    "$workspace/openapi-kt" generate --validation --input "$project/wide.json" --output "$project/build/generated-limits/map" --package-name org.example.mapwide --model-name-suffix Dto --model-style "$style" --check
  fi
  if [[ "$checked" == off ]]; then
    if "$workspace/openapi-kt" validate --input tests/fixtures/http/map-unique.json > "$project/rejected.log" 2>&1; then
      echo 'Validation-off accepted MAP uniqueItems' >&2; exit 1
    fi
    rg -q E_SCHEMA_UNSUPPORTED "$project/rejected.log"
  fi
  cat >> "$project/build.gradle.kts" <<'GRADLE'
tasks.register<JavaExec>("runMapUniqueContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
GRADLE
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
    echo "PREPARED MAP UNIQUE $style $checked: $project"
    continue
  fi
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runMapUniqueContracts
 done
done
echo "Fresh MAP uniqueness projects: $workspace"
