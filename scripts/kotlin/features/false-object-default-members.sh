#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-object-default-members-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseObjectDefaultMembersContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.falseobjectdefaults; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.falseobjectdefaultsoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      suffix=; [[ "$mode" == base ]] || suffix=Dto
      args=(--input tests/fixtures/http/false-object-default-members.json --output "$output" --package-name "$package" --api-client-name FalseObjectDefaultsClient --model-name-suffix "$suffix" --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    python3 - "$project" "$checked" <<'PYWIDE'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on';ref=lambda name,**extra:{'$ref':'#/components/schemas/'+name,**extra}
nullable={f'p{i:03d}':ref('Maybe',default=None) for i in range(240)}
values={key:None for key in nullable}
schemas={'Blocked':False,'Maybe':ref('Blocked',nullable=True,default=None),'WideOptional':{'properties':{key:ref('Blocked') for key in nullable},'default':{}},'WideNullable':{'properties':nullable,'default':values},'WideRequired':{'required':list(nullable),'properties':nullable,'default':values}}
if checked:
    schemas['WideOptional']['maxProperties']=0
    schemas['WideRequired']['minProperties']=240
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
PYWIDE
    wide_args=(--input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.falseobjectdefaultswide --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}"
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/FalseObjectDefaultMembersContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
import json
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts')
optimized=optimized.replace('org.example.falseobjectdefaults.FalseObjectDefaultsClient','org.example.falseobjectdefaultsoptimized.FalseObjectDefaultsClient')
for name in json.loads(Path('tests/fixtures/http/false-object-default-members.json').read_text())['components']['schemas']:
    optimized=optimized.replace('org.example.falseobjectdefaults.models.'+name+' as ', 'org.example.falseobjectdefaultsoptimized.models.'+name+'Dto as ')
    optimized=optimized.replace('org.example.falseobjectdefaults.models.`'+name+' default` as ', 'org.example.falseobjectdefaultsoptimized.models.`'+name+'Dto default` as ')
(opt/'FalseObjectDefaultMembersContracts.kt').write_text(optimized)
wide=r"""
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
fun wideFalseObjectDefaults() {
    val optional=org.example.falseobjectdefaultswide.models.WideOptionalDto.default()
    val nullable=org.example.falseobjectdefaultswide.models.WideNullableDto.default()
    val required=org.example.falseobjectdefaultswide.models.WideRequiredDto.default()
    check(org.example.falseobjectdefaultswide.models.WideRequiredDto.default() !== required)
    val descriptor=kotlinx.serialization.serializer<org.example.falseobjectdefaultswide.models.WideRequiredDto>().descriptor
    for(index in 0 until 240) check(!descriptor.isElementOptional(index))
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=kotlinx.serialization.json.Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(codec.encodeToString(optional)=="{}")
        check(codec.parseToJsonElement(codec.encodeToString(nullable)).jsonObject.size == if(defaults && nulls) 240 else 0)
        if(CHECKED && !nulls) {
            check(runCatching { codec.encodeToString(required) }.exceptionOrNull() is kotlinx.serialization.SerializationException)
        } else {
            check(codec.parseToJsonElement(codec.encodeToString(required)).jsonObject.size == if(nulls) 240 else 0)
        }
        for(index in 0 until 240) {
            val field="p"+index.toString().padStart(3,'0')
            check(runCatching { codec.decodeFromString<org.example.falseobjectdefaultswide.models.WideOptionalDto>("{\"$field\":null}") }.isFailure)
            check(runCatching { codec.decodeFromString<org.example.falseobjectdefaultswide.models.WideNullableDto>("{\"$field\":true}") }.isFailure)
        }
    }
}
""".replace('CHECKED',checked)
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text('package org.example.contracts\nimport kotlinx.serialization.json.jsonObject\n'+wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { wideFalseObjectDefaults(); falseObjectDefaultMembersContracts(); org.example.optimizedcontracts.falseObjectDefaultMembersContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runFalseObjectDefaultMembersContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED FALSE OBJECT DEFAULT MEMBERS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runFalseObjectDefaultMembersContracts
    echo "FRESH FALSE OBJECT DEFAULT MEMBERS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh false object default member projects: $workspace"
