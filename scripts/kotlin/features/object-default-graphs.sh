#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/object-default-graphs-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ObjectDefaultGraphsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$checked" <<'PYTHON'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/object-default-graphs.json').read_text())
if checked:
    for name in ['Text']:
        d['components']['schemas'][name]['minLength']=2
    for name,field in [('Node','name'),('Base','label')]:
        if name in d['components']['schemas']:
            d['components']['schemas'][name]['properties'][field]['minLength']=2
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
schemas={'Text':{'type':'string'}}
for i in reversed(range(15)):
    child={'$ref':'#/components/schemas/'+('C'+str(i+1) if i<14 else 'Text')}
    node={'type':'array','items':child} if i%2==0 else {'type':'object','additionalProperties':child}
    if checked:node['minItems' if i%2==0 else 'minProperties']=0
    schemas['C'+str(i)]=node
schemas['Rows']={'type':'array','items':{'type':'object','additionalProperties':{'type':'string'}}}
schemas['Wide']={'properties':{'cfg':{'properties':{'p'+str(i):{'$ref':'#/components/schemas/Rows'} for i in range(240)},'default':{}}}}
schemas['Deep']={'properties':{'cfg':{'properties':{'value':{'$ref':'#/components/schemas/C0'}},'default':{'value':[]}}}}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}})+'\n')
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
    check(org.example.graphlimits.models.WideDto().cfg!!.p239 == null)
    check(org.example.graphlimits.models.DeepDto().cfg!!.value!!.isEmpty())
    val optimized = org.example.graphoptimized.models.HolderDto("token")
    check(optimized.cfg!!.rows!!.first()!!.getValue("a") == "first")
    check(optimized.cfg !== org.example.graphoptimized.models.HolderDto("token").cfg)
    objectDefaultGraphsContracts('''+str(checked).lower()+''')
}
''')
PYTHON
  flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.defaultgraphs --model-name-suffix Dto --model-style "$style")
  validation=()
  if [[ "$checked" == on ]]; then flags+=(--validation); validation+=(--validation); fi
  "$workspace/openapi-kt" generate "${flags[@]}"
  "$workspace/openapi-kt" generate "${flags[@]}" --check
  for mode in optimized limits; do
   input="$project/input.json"; package=org.example.graphoptimized; output="$project/build/generated-options"
   options=()
   if [[ "$mode" == optimized ]]; then
    options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
   else
    input="$project/limits.json"; package=org.example.graphlimits; output="$project/build/generated-limits"
   fi
   args=(--input "$input" --output "$output" --package-name "$package" --model-name-suffix Dto --model-style "$style")
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
  done
  cat >> "$project/build.gradle.kts" <<'GRADLE'
tasks.register<JavaExec>("runObjectDefaultGraphsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
GRADLE
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
   echo "PREPARED OBJECT DEFAULT GRAPHS $style $checked: $project"
   continue
  fi
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runObjectDefaultGraphsContracts
 done
done
echo "Fresh object-default graph projects: $workspace"
