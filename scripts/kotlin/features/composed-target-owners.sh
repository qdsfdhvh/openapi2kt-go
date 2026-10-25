#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/composed-target-owners-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for owner_role in allof-owner union-owner plain-leaf composed-leaf; do
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$owner_role/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ComposedTargetOwnersContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    python3 - "$project" "$checked" "$owner_role" <<'PYMAIN'
from pathlib import Path
import sys,json
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
d=json.loads(Path('tests/fixtures/http/composed-target-owners-'+sys.argv[3]+'.json').read_text())
owner=d['components']['schemas']['Owner'];owner=owner.get('allOf',owner.get('oneOf',[owner]))[0]
if checked=='true':owner['properties']['payload']['allOf'][0]['properties']['node']['properties']['count']['minimum']=1
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
nullable=json.loads(Path('tests/fixtures/http/nullable-composed-target-owners-'+sys.argv[3]+'.json').read_text())
owner=nullable['components']['schemas']['Owner'];owner=owner.get('allOf',owner.get('oneOf',[owner]))[0]
if checked=='true':owner['properties']['payload']['allOf'][0]['properties']['node']['properties']['count']['minimum']=1
(p/'nullable.json').write_text(json.dumps(nullable,indent=2)+'\n')
source=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableComposedTargetOwnersContracts.kt').read_text()
if sys.argv[3]=='union-owner':source=source.replace('OwnerDto.Payload as ValueDto','OwnerDto.Variant1.Payload as ValueDto')
(p/'src/main/kotlin/org/example/contracts/NullableComposedTargetOwnersContracts.kt').write_text(source)
(opt_source:=p/'src/main/kotlin/org/example/optimizedcontracts').mkdir(parents=True,exist_ok=True)
(opt_source/'NullableComposedTargetOwnersContracts.kt').write_text(source.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.composedtargetownersnullable','org.example.composedtargetownersoptimizednullable'))

wide={'openapi':'3.1.0','components':{'schemas':{'Owner':{'properties':{'node':{'properties':{'p'+str(i):{'type':'integer'} for i in range(239)}}}},'Hops':{'properties':{'v':{'$ref':'#/components/schemas/Owner/properties/node'}}},'Bridge':{'$ref':'#/components/schemas/Hops/properties/v'},'Consumer':{'allOf':[{'$ref':'#/components/schemas/Bridge'},{'properties':{'extra':{'type':'string'}}}]}}}}
wide['components']['schemas']['Owner']['properties']['node']={'allOf':[wide['components']['schemas']['Owner']['properties']['node']]}
owner=wide['components']['schemas']['Owner'];path='Owner/properties/'
if sys.argv[3] in ('allof-owner','composed-leaf'):wide['components']['schemas']['Owner']={'allOf':[owner]};path='Owner/allOf/0/properties/'
if sys.argv[3]=='union-owner':wide['components']['schemas']['Owner']={'oneOf':[owner,{'properties':{'other':{'type':'boolean'}}}]};path='Owner/oneOf/0/properties/'
if sys.argv[3] in ('plain-leaf','composed-leaf'):
 wide['components']['schemas']['Root']={'oneOf':[{'$ref':'#/components/schemas/Owner'},{'$ref':'#/components/schemas/OtherOwner'}],'discriminator':{'propertyName':'kind'}}
 wide['components']['schemas']['OtherOwner']={'properties':{'other':{'type':'boolean'}}}
wide['components']['schemas']['Hops']['properties']['v']['$ref']='#/components/schemas/'+path+'node'
(p/'wide.json').write_text(json.dumps(wide)+'\n')


consumer=(p/'src/main/kotlin/org/example/contracts/ComposedTargetOwnersContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'ComposedTargetOwnersContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.composedtargetowners','org.example.composedtargetownersoptimized'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { check(org.example.composedtargetownerswide.models.ConsumerDto(p238=238,extra="wide").p238==238); composedTargetOwnersContracts('+checked+'); nullableComposedTargetOwnersContracts('+checked+'); org.example.optimizedcontracts.composedTargetOwnersContracts('+checked+'); org.example.optimizedcontracts.nullableComposedTargetOwnersContracts('+checked+') }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
kotlin.sourceSets.getByName("main").kotlin.srcDirs("build/generated-nullable", "build/generated-limits-nullable")
tasks.register<JavaExec>("runComposedTargetOwnersContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    for mode in base optimized wide; do
      package=org.example.composedtargetowners; output="$project/build/generated"; options=(); input="$project/input.json"
      if [[ "$mode" == optimized ]]; then
        package=org.example.composedtargetownersoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      elif [[ "$mode" == wide ]]; then
        package=org.example.composedtargetownerswide;output="$project/build/generated-options";input="$project/wide.json"
      fi
      args=(--input "$input" --output "$output" --package-name "$package" --api-client-name BridgeClient --model-name-suffix Dto --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
      if [[ "$mode" != wide ]]; then
        "$workspace/openapi-kt" generate --input "$project/nullable.json" --output "${output}-nullable" --package-name "${package}nullable" --api-client-name BridgeClient --model-name-suffix Dto --model-style "$style" "${validation[@]}" "${options[@]}"
        "$workspace/openapi-kt" generate --input "$project/nullable.json" --output "${output}-nullable" --package-name "${package}nullable" --api-client-name BridgeClient --model-name-suffix Dto --model-style "$style" "${validation[@]}" "${options[@]}" --check
      fi
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED INTERNAL DTO INTERSECTIONS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runComposedTargetOwnersContracts
    echo "FRESH INTERNAL DTO INTERSECTIONS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh bridge intersection projects: $workspace"

done
