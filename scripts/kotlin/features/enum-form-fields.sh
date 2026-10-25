#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/enum-form-fields-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/EnumFormFieldsMediaContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/enum-form-fields.json').read_text())
(p/'input.json').write_text(json.dumps(d,indent=2,ensure_ascii=False)+'\n')
fields={'code':{'type':'string','enum':['c'+str(i) for i in range(240)]},'native':{'type':'integer','enum':list(range(240))}}
wide={'openapi':'3.1.0','paths':{'/wide':{'post':{'operationId':'wide','requestBody':{'required':True,'content':{'application/x-www-form-urlencoded':{'schema':{'type':'object','properties':fields}}}},'responses':{'204':{}}}}}}
(p/'wide.json').write_text(json.dumps(wide)+'\n')
consumer=p/'src/main/kotlin/org/example/contracts/EnumFormFieldsMediaContracts.kt'

(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
 check(org.example.enumformwide.api.models.WideInputDto(code=org.example.enumformwide.api.models.WideInputDto.Code.fromSerialName("c239"),native=239).code?.serialName == "c239")
 enumFormFieldsMediaContracts('''+str(checked).lower()+''')
 optimizedEnumFormFieldsMediaContracts('''+str(checked).lower()+''')
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runEnumFormFieldsMediaContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
    for mode in baseline optimized wide; do
      input="$project/input.json"
      output="$project/build/generated"
      package=org.example.enumform
      options=()
      if [[ "$mode" == optimized ]]; then
        output="$project/build/generated-limits"
        package=org.example.enumformoptimized
        options=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      elif [[ "$mode" == wide ]]; then
        input="$project/wide.json"
        output="$project/build/generated-options"
        package=org.example.enumformwide
      fi
      "$workspace/openapi-kt" generate --input "$input" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input "$input" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
    done
    python3 - "$project" <<'PYOWNER'
from pathlib import Path
import re,sys,json
p=Path(sys.argv[1]);source=p/'src/main/kotlin/org/example/contracts/EnumFormFieldsMediaContracts.kt'
consumer=source.read_text().replace('org.example.enumform','org.example.enumformoptimized').replace('EnumFormFields','OptimizedEnumFormFields').replace('enumFormFields','optimizedEnumFormFields')
aliases={}
for path in (p/'build/generated-limits').rglob('*.kt'):
 text=path.read_text();package=re.search(r'^package ([\w.]+)',text,re.M)
 if not package:continue
 for name,target in re.findall(r'^public typealias (\w+) = ([\w.]+)',text,re.M):
  aliases[package[1]+'.'+name]=target if '.' in target else package[1]+'.'+target
bindings={}
for name in ['SubmitInputDto','OptionalInputDto','EventsInputDto','OptionalEventsInputDto']:
 owner='org.example.enumformoptimized.api.models.'+name;seen=set()
 while owner in aliases:
  assert owner not in seen;seen.add(owner);owner=aliases[owner]
 bindings[name]=owner
 for member in ['Code','OptionalCode']:
  consumer=re.sub(r'\b'+re.escape(name+'.'+member)+r'\b',owner+'.'+member,consumer)
(p/'src/main/kotlin/org/example/contracts/OptimizedEnumFormFieldsMediaContracts.kt').write_text(consumer)
(p/'optimized-enum-owner-bindings.json').write_text(json.dumps(bindings,indent=2)+'\n')
PYOWNER
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runEnumFormFieldsMediaContracts
    echo "FRESH ENUM FORM FIELDS PROFILE $style $checked PASS: $project"
  done
done
