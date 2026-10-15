#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/collection-owned-dto-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for profile in data-off data-on plain-off plain-on all-on; do
  style=data_class; checked=off
  case "$profile" in plain-*) style=plain_class;; esac
  case "$profile" in *-on) checked=on;; esac
  project="$workspace/$profile"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/CollectionOwnedDTOContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$checked" "$style" <<'PY'
import json,sys
from pathlib import Path
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'; style=sys.argv[3]
doc=json.loads(Path('tests/fixtures/http/schema-collection-owned-dtos.json').read_text())
def checks(value):
    if isinstance(value,dict):
        for name,field in value.get('properties',{}).items():
            if name in ('id','name') and field.get('type')=='string':field['minLength']=2
        for child in value.values():checks(child)
    elif isinstance(value,list):
        for child in value:checks(child)
if checked:checks(doc)
(p/'input.json').write_text(json.dumps(doc))
ref={'$ref':'#/components/schemas/Owner/items'}
value=ref
for _ in range(46):value={'type':'array','items':value}
probes={'openapi':'3.1.0','components':{'schemas':{
 'Owner':{'type':'array','items':{'properties':{'id':{'type':'string'}},'required':['id']}},
 'Wide':{'properties':{f'p{i}':ref for i in range(240)}},
 'Deep':{'properties':{'value':value}}
}}}
(p/'limits.json').write_text(json.dumps(probes))
copy='' if style=='plain_class' else '''
    val original=org.example.collectiondtos.models.RootRowsDtoInner("valid")
    check(original.copy(id="copy").id=="copy")
'''
if checked and style!='plain_class':copy+='    check(runCatching { original.copy(id="") }.isFailure)\n'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main()=runBlocking {
'''+copy+'''    collectionOwnedDTOContracts('''+str(checked).lower()+''')
}
''')
build=p/'build.gradle.kts'
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runCollectionOwnedDtoContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  flags=(--model-style "$style")
  if [[ "$checked" == on ]]; then flags+=(--validation); fi
  if [[ "$profile" == all-on ]]; then
    flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models)
  fi
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.collectiondtos --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.collectiondtos --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.collectiondtolimits --model-name-suffix Dto "${flags[@]}"
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runCollectionOwnedDtoContracts
  echo "FRESH COLLECTION OWNED DTO PROFILE $profile PASS: $project"
done
echo "Fresh collection-owned DTO projects: $workspace"
