#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/unformatted-native-factories-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnformattedNativeFactoriesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
ratio={'type':'number','enum':[1.5,2.5]}
wide={'properties':{'p'+str(i):{'$ref':'#/components/schemas/Ratio'} for i in range(120)},'required':['p'+str(i) for i in range(120)]}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'Ratio':ratio,'Wide':wide}}})+'\n')
nullable={'Code':{'type':'integer','nullable':True,'enum':[1,2],'default':None},'Narrow':{'$ref':'#/components/schemas/Code','nullable':False,'default':2},'WithNull':{'type':'integer','nullable':True,'enum':list(range(239))+[None],'default':None}}
(p/'nullable.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':nullable}})+'\n')
PYINPUT
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.unformatted; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.unformattedoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      args=(--input tests/fixtures/http/unformatted-native-factories.json --output "$output" --package-name "$package" --api-client-name UnformattedFactoryClient --model-name-suffix Dto --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-options/limits" --package-name org.example.unformattedlimits --model-name-suffix Dto --model-style "$style" "${validation[@]}"
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-options/limits" --package-name org.example.unformattedlimits --model-name-suffix Dto --model-style "$style" "${validation[@]}" --check
    if [[ "$checked" == on ]]; then
      "$workspace/openapi-kt" generate --input "$project/nullable.json" --output "$project/build/generated-options/nullable" --package-name org.example.unformattednullable --model-name-suffix Dto --model-style "$style" --validation
      "$workspace/openapi-kt" generate --input "$project/nullable.json" --output "$project/build/generated-options/nullable" --package-name org.example.unformattednullable --model-name-suffix Dto --model-style "$style" --validation --check
    fi
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
nullable_checks=''
if checked=='true':
 nullable_checks='check(org.example.unformattednullable.models.`CodeDto default`() == null); check(org.example.unformattednullable.models.`NarrowDto default`() == 2); check(org.example.unformattednullable.models.`WithNullDto default`() == null); '
wide_check='check(org.example.unformattedlimits.models.WideDto('+','.join(['1.5']*120)+').p119 == 1.5); '

consumer=(p/'src/main/kotlin/org/example/contracts/UnformattedNativeFactoriesContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'UnformattedNativeFactoriesContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.unformatted','org.example.unformattedoptimized'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { '+nullable_checks+wide_check+'unformattedNativeFactoriesContracts('+checked+'); org.example.optimizedcontracts.unformattedNativeFactoriesContracts('+checked+') }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runUnformattedNativeFactoriesContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED UNFORMATTED NATIVE FACTORIES $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runUnformattedNativeFactoriesContracts
    echo "FRESH UNFORMATTED NATIVE FACTORIES PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh unformatted native factory projects: $workspace"
