#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/union-allof-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class;do
 for checked in off on;do
  project="$workspace/$style-$checked";mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnionAllOfContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  flags=(--model-style "$style");if [[ "$checked" == on ]];then flags+=(--validation);fi
  "$workspace/openapi-kt" generate --input tests/fixtures/http/union-allof.json --output "$project/build/generated" --package-name org.example.unionallof --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input tests/fixtures/http/union-allof.json --output "$project/build/generated" --package-name org.example.unionallof --model-name-suffix Dto "${flags[@]}" --check
  python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
def choice(field,n): return {'oneOf':[{'type':'object','properties':{field:{'type':'string','const':field+str(i)}}} for i in range(n)]}
product={'allOf':[choice('kind',15),choice('mode',16)]}
wide={'allOf':[{'type':'object','properties':{'p'+str(i):{'type':'boolean'} for i in range(239)},'required':['p'+str(i) for i in range(239)]},choice('kind',2)]}
value={'allOf':[choice('kind',2)]}
# Short synthetic names keep the same 45 object edges within host class-file
# filename limits; source depth, union branches and constructor caps do not change.
for i in range(45): value={'type':'object','properties':{'n'+str(i):value}}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'ProductLimit':product,'WideLimit':wide,'DeepLimit':value}}}))
PYLIMITS
  "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.unionalloflimits --model-name-suffix Dto "${flags[@]}"
  python3 - "$project" "$style" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else '''val value = org.example.unionallof.models.ComposedDto.Note("copy",1,"body")
    check(value.copy(body="updated").body=="updated")'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main()=runBlocking {
    '''+copy+'''
    unionAllOfContracts()
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runUnionAllOfContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runUnionAllOfContracts
  echo "FRESH UNION ALLOF PROFILE $style $checked PASS: $project"
 done
done
echo "Fresh union/allOf native projects: $workspace"
