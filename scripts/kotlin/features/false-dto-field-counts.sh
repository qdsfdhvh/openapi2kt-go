#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-dto-field-counts-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseDTOFieldCountsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.falsedtocounts; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.falsedtocountsoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      suffix=; [[ "$mode" == base ]] || suffix=Dto
      args=(--input tests/fixtures/http/false-dto-field-counts.json --output "$output" --package-name "$package" --api-client-name FalseDTOFieldCountsClient --model-name-suffix "$suffix" --model-style "$style")
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
if checked:
    schemas['WideTarget']['minProperties']=240
    schemas['WideTarget']['maxProperties']=240
    schemas['WideConsumer']['maxProperties']=240
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
PYWIDE
    wide_args=(--input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.falsedtocountswide --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}"
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/FalseDTOFieldCountsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
import json
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts')
optimized=optimized.replace('org.example.falsedtocounts.FalseDTOFieldCountsClient','org.example.falsedtocountsoptimized.FalseDTOFieldCountsClient')
for name in json.loads(Path('tests/fixtures/http/false-dto-field-counts.json').read_text())['components']['schemas']:
    optimized=optimized.replace('org.example.falsedtocounts.models.'+name+' as ', 'org.example.falsedtocountsoptimized.models.'+name+'Dto as ')
    optimized=optimized.replace('org.example.falsedtocounts.models.`'+name+' default` as ', 'org.example.falsedtocountsoptimized.models.`'+name+'Dto default` as ')
(opt/'FalseDTOFieldCountsContracts.kt').write_text(optimized)
wide=r"""
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
fun wideFalseDTOFieldCounts() {
    val first=org.example.falsedtocountswide.models.WideConsumerDto()
    val second=org.example.falsedtocountswide.models.WideConsumerDto()
    check(first.p000 !== second.p000 && first.p239 !== second.p239)
    val member=org.example.falsedtocountswide.models.WideMemberDto()
    check(member.target != null && member.target?.p239 == null)
    check(org.example.falsedtocountswide.models.WideMemberDto().target !== member.target)
    val descriptor=kotlinx.serialization.serializer<org.example.falsedtocountswide.models.WideRequiredDto>().descriptor
    for(index in 0 until 240) check(!descriptor.isElementOptional(index))
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=kotlinx.serialization.json.Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(codec.parseToJsonElement(codec.encodeToString(first)).jsonObject.size == if(defaults || first.p000 != second.p000) 240 else 0)
        val fresh=codec.decodeFromString<org.example.falsedtocountswide.models.WideConsumerDto>("{}")
        check(fresh.p000 !== first.p000 && fresh.p239 !== first.p239)
        check(runCatching { codec.decodeFromString<org.example.falsedtocountswide.models.WideRequiredDto>("{}") }.isFailure)
        if(!defaults || !nulls) {
            check(runCatching { codec.encodeToString(member.target!!) }.exceptionOrNull() is kotlinx.serialization.SerializationException)
        } else {
            check(codec.parseToJsonElement(codec.encodeToString(member.target!!)).jsonObject.size == 240)
        }
    }
}
""".replace('CHECKED',checked)
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text('package org.example.contracts\nimport kotlinx.serialization.json.jsonObject\n'+wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { wideFalseDTOFieldCounts(); falseDTOFieldCountsContracts(); org.example.optimizedcontracts.falseDTOFieldCountsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runFalseDTOFieldCountsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED FALSE DTO FIELD COUNTS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runFalseDTOFieldCountsContracts
    echo "FRESH FALSE DTO FIELD COUNTS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh false DTO field count projects: $workspace"
