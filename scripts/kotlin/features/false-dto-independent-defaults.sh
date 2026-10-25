#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-dto-independent-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseDTOIndependentDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.falsedtoindependent; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.falsedtoindependentoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      suffix=; [[ "$mode" == base ]] || suffix=Dto
      args=(--input tests/fixtures/http/false-dto-independent-defaults.json --output "$output" --package-name "$package" --api-client-name FalseDTOIndependentDefaultsClient --model-name-suffix "$suffix" --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    python3 - "$project" "$checked" <<'PYWIDE'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on';ref=lambda name,**extra:{'$ref':'#/components/schemas/'+name,**extra}
keys=[f'p{i:03d}' for i in range(240)]
props={key:ref('Maybe') for key in keys}
values={key:None for key in keys}
schemas={'Blocked':False,'Maybe':ref('Blocked',nullable=True),'Target':{'required':['value'],'properties':{'value':ref('Maybe'),'blocked':ref('Blocked')}},'WideTarget':{'required':keys,'properties':props},'WideConsumer':{'properties':{key:ref('Target',default={'value':None}) for key in keys}},'WideRequired':{'required':keys,'properties':{key:ref('Target',default={'value':None}) for key in keys}},'WideMember':{'properties':{'target':ref('WideTarget',default=values)}}}
schemas['Target']['default']={'value':None}
schemas['WideTarget']['default']=values
if checked:
    schemas['WideConsumer']['maxProperties']=240
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
PYWIDE
    wide_args=(--input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.falsedtoindependentwide --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}"
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/FalseDTOIndependentDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
import json
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts')
optimized=optimized.replace('org.example.falsedtoindependent.FalseDTOIndependentDefaultsClient','org.example.falsedtoindependentoptimized.FalseDTOIndependentDefaultsClient')
for name in json.loads(Path('tests/fixtures/http/false-dto-independent-defaults.json').read_text())['components']['schemas']:
    optimized=optimized.replace('org.example.falsedtoindependent.models.'+name+' as ', 'org.example.falsedtoindependentoptimized.models.'+name+'Dto as ')
    optimized=optimized.replace('org.example.falsedtoindependent.models.`'+name+' default` as ', 'org.example.falsedtoindependentoptimized.models.`'+name+'Dto default` as ')
(opt/'FalseDTOIndependentDefaultsContracts.kt').write_text(optimized)
wide=r"""
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
fun wideFalseDTOIndependentDefaults() {
    val first=org.example.falsedtoindependentwide.models.WideConsumerDto()
    val second=org.example.falsedtoindependentwide.models.WideConsumerDto()
    check(first.p000 !== second.p000 && first.p239 !== second.p239)
    val member=org.example.falsedtoindependentwide.models.WideMemberDto()
    check(member.target != null && member.target?.p239 == null)
    check(org.example.falsedtoindependentwide.models.WideMemberDto().target !== member.target)
    val descriptor=kotlinx.serialization.serializer<org.example.falsedtoindependentwide.models.WideRequiredDto>().descriptor
    for(index in 0 until 240) check(!descriptor.isElementOptional(index))
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=kotlinx.serialization.json.Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(codec.parseToJsonElement(codec.encodeToString(first)).jsonObject.size == if(defaults || first.p000 != second.p000) 240 else 0)
        val fresh=codec.decodeFromString<org.example.falsedtoindependentwide.models.WideConsumerDto>("{}")
        check(fresh.p000 !== first.p000 && fresh.p239 !== first.p239)
        check(runCatching { codec.decodeFromString<org.example.falsedtoindependentwide.models.WideRequiredDto>("{}") }.isFailure)
        check(codec.encodeToString(member).isNotEmpty())
    }
}
""".replace('CHECKED',checked)
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text('package org.example.contracts\nimport kotlinx.serialization.json.jsonObject\n'+wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { wideFalseDTOIndependentDefaults(); falseDTOIndependentDefaultsContracts(); org.example.optimizedcontracts.falseDTOIndependentDefaultsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runFalseDTOIndependentDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED FALSE DTO INDEPENDENT DEFAULTS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runFalseDTOIndependentDefaultsContracts
    echo "FRESH FALSE DTO INDEPENDENT DEFAULTS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh false DTO independent default projects: $workspace"
