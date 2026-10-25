#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/object-collection-factories-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ObjectCollectionFactoriesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/object-collection-factories.json').read_text())
def bounds(v):
 if isinstance(v,dict):
  field=v.get('properties',{}).get('value')
  if field is not None and checked:field['minLength']=1
  for child in v.values():bounds(child)
 elif isinstance(v,list):
  for child in v:bounds(child)
bounds(d);(p/'input.json').write_text(json.dumps(d)+'\n')
leaf={'properties':{'p'+str(i):{'type':'string',**({'minLength':1} if checked else {})} for i in range(240)}}
deep={'properties':{'value':{'type':'string',**({'minLength':1} if checked else {})}}}
for i in range(15):deep={'type':'array','items':deep}
deep['default']=[]
wide={'type':'array','items':leaf,'default':[{'p239':'last'}]}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'Wide':wide,'Deep':deep}}})+'\n')
PYINPUT
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.objectfactory; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.objectfactoryoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      args=(--input "$project/input.json" --output "$output" --package-name "$package" --api-client-name ObjectFactoryClient --model-name-suffix Dto --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-options" --package-name org.example.factorylimits --model-name-suffix Dto --model-style "$style" "${validation[@]}"
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-options" --package-name org.example.factorylimits --model-name-suffix Dto --model-style "$style" "${validation[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/ObjectCollectionFactoriesContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'ObjectCollectionFactoriesContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.objectfactory','org.example.objectfactoryoptimized'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nimport org.example.factorylimits.models.`WideDto default`\nimport org.example.factorylimits.models.`DeepDto default`\nfun main() = runBlocking { check(`WideDto default`().single().p239 == "last"); check(`DeepDto default`().isEmpty()); objectCollectionFactoriesContracts('+checked+'); org.example.optimizedcontracts.objectCollectionFactoriesContracts('+checked+') }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runObjectCollectionFactoriesContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED OBJECT COLLECTION FACTORIES $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runObjectCollectionFactoriesContracts
    echo "FRESH OBJECT COLLECTION FACTORIES PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh object collection factory projects: $workspace"
