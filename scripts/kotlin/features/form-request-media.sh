#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/form-request-media-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FormRequestMediaContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/form-request-media.json').read_text())
if checked:
 for path in d['paths'].values():
  for op in path.values():
   for media in op['requestBody']['content'].values():
    text=media['schema'].get('properties',{}).get('text')
    if text is not None:text['minLength']=1
(p/'input.json').write_text(json.dumps(d,indent=2,ensure_ascii=False)+'\n')
fields={'p'+str(i):{'type':'string',**({'minLength':1} if checked else {})} for i in range(240)}
wide={'openapi':'3.1.0','paths':{'/wide':{'post':{'operationId':'wide','requestBody':{'required':True,'content':{'application/x-www-form-urlencoded':{'schema':{'type':'object','properties':fields,'required':list(fields)}}}},'responses':{'204':{}}}}}}
(p/'wide.json').write_text(json.dumps(wide)+'\n')
consumer=p/'src/main/kotlin/org/example/contracts/FormRequestMediaContracts.kt'
(p/'src/main/kotlin/org/example/contracts/OptimizedFormRequestMediaContracts.kt').write_text(consumer.read_text().replace('org.example.formmedia','org.example.formmediaoptimized').replace('FormRequest','OptimizedFormRequest').replace('formRequest','optimizedFormRequest'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
 check(org.example.formmediawide.api.models.WideInputDto('''+','.join(['"value"']*240)+''').p239 == "value")
 formRequestMediaContracts('''+str(checked).lower()+''')
 optimizedFormRequestMediaContracts('''+str(checked).lower()+''')
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runFormRequestMediaContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
    for mode in baseline optimized wide; do
      input="$project/input.json"
      output="$project/build/generated"
      package=org.example.formmedia
      options=()
      if [[ "$mode" == optimized ]]; then
        output="$project/build/generated-limits"
        package=org.example.formmediaoptimized
        options=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      elif [[ "$mode" == wide ]]; then
        input="$project/wide.json"
        output="$project/build/generated-options"
        package=org.example.formmediawide
      fi
      "$workspace/openapi-kt" generate --input "$input" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input "$input" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
    done
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runFormRequestMediaContracts
    echo "FRESH FORM REQUEST MEDIA PROFILE $style $checked PASS: $project"
  done
done
