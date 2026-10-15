#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/boolean-root-nullable-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for fixture in boolean-root-nullable recursive-json-reference-graph;do
 for style in data_class plain_class;do
  for checked in off on;do
  project="$workspace/$fixture-$style-$checked";mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BooleanRootNullableContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  flags=(--model-style "$style");if [[ "$checked" == on ]];then flags+=(--validation);fi
  "$workspace/openapi-kt" generate --input "tests/fixtures/http/$fixture.json" --output "$project/build/generated" --package-name org.example.booleanrootnullable --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "tests/fixtures/http/$fixture.json" --output "$project/build/generated" --package-name org.example.booleanrootnullable --model-name-suffix Dto "${flags[@]}" --check
  if [[ "$fixture" == recursive-json-reference-graph ]];then
   python3 - "$project" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
cover={'oneOf':[{'type':'null'},{'type':'string'},{'type':'boolean'},{'type':'number'},{'type':'array','items':{'$ref':r+'Value'}},{'type':'object','additionalProperties':{'$ref':r+'Value'}}]}
leaf={'$ref':r+'Nullable','nullable':False}
for _ in range(43): leaf={'type':'array','items':leaf}
s={'Value':cover,'Nullable':{'$ref':r+'Value','nullable':True},'Wide':{'type':'object','required':['p'+str(i) for i in range(240)],'properties':{'p'+str(i):{'$ref':r+'Nullable'} for i in range(240)}},'Deep':{'type':'object','properties':{'value':leaf}}}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s}}))
PY
   "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.recursivegraphlimits --model-name-suffix Dto "${flags[@]}"
  fi
  python3 - "$project" "$style" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);extra='' if sys.argv[2]=='plain_class' else '''val value=org.example.booleanrootnullable.models.HolderDto(canonical=kotlinx.serialization.json.JsonNull,inherited=null,narrowed=kotlinx.serialization.json.JsonNull,rows=emptyList(),mapping=emptyMap())
    check(value.copy(canonical=kotlinx.serialization.json.JsonPrimitive("copy")).canonical==kotlinx.serialization.json.JsonPrimitive("copy"))'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main()=runBlocking {
    '''+extra+'''
    booleanRootNullableContracts()
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''\ntasks.register<JavaExec>("runBooleanRootNullableContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runBooleanRootNullableContracts
  echo "FRESH BOOLEAN ROOT NULLABLE PROFILE $fixture $style $checked PASS: $project"
  done
 done
done
echo "Fresh Boolean root native projects: $workspace"
