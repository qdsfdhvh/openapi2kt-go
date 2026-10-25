#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-inline-collection-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseInlineCollectionDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.falseinlinedefaults; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.falseinlinedefaultsoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      suffix=; [[ "$mode" == base ]] || suffix=Dto
      args=(--input tests/fixtures/http/false-inline-collection-defaults.json --output "$output" --package-name "$package" --api-client-name FalseInlineDefaultsClient --model-name-suffix "$suffix" --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    python3 - "$project" "$checked" <<'PYWIDE'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on';ref=lambda name:{'$ref':'#/components/schemas/'+name}
leaf=ref('Blocked');valid='[]';invalid='[null]'
for i in range(14,-1,-1):
    if i%2==0: leaf={'type':'array','items':leaf}
    else: leaf={'additionalProperties':leaf}
    if checked: leaf['minItems' if i%2==0 else 'minProperties']=0
    if i<14:
        valid=('['+valid+']') if i%2==0 else ('{"k":'+valid+'}')
        invalid=('['+invalid+']') if i%2==0 else ('{"k":'+invalid+'}')
rows={'type':'array','items':ref('Blocked')}
rows['default']=[]
if checked: rows['maxItems']=0
schemas={'Blocked':False,'Rows':{'type':'array','items':ref('Blocked')},'Wide':{'properties':{f'p{i:03d}':rows for i in range(240)}},'Deep':leaf,'DeepHolder':{'required':['deep'],'properties':{'deep':{**leaf,'default':json.loads(valid)}}}}
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
large={'Blocked':False,'Large':{'properties':{'rows':{'type':'array','items':{'$ref':'#/components/schemas/Blocked','nullable':True},'default':[None]*6553}}}}
(p/'large.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':large}}))
(p/'deep-values.json').write_text(json.dumps({'valid':valid,'invalid':invalid}))
PYWIDE
    wide_args=(--input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.falseinlinedefaultswide --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}"
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}" --check
    large_args=(--input "$project/large.json" --output "$project/build/generated-large" --package-name org.example.falseinlinedefaultslarge --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${large_args[@]}" "${validation[@]}"
    "$workspace/openapi-kt" generate "${large_args[@]}" "${validation[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/FalseInlineCollectionDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
import json
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts')
optimized=optimized.replace('org.example.falseinlinedefaults.FalseInlineDefaultsClient','org.example.falseinlinedefaultsoptimized.FalseInlineDefaultsClient')
for name in json.loads(Path('tests/fixtures/http/false-inline-collection-defaults.json').read_text())['components']['schemas']:
    optimized=optimized.replace('org.example.falseinlinedefaults.models.'+name+' as ', 'org.example.falseinlinedefaultsoptimized.models.'+name+'Dto as ')
(opt/'FalseInlineCollectionDefaultsContracts.kt').write_text(optimized)
deep=json.loads((p/'deep-values.json').read_text())
wide=r"""
fun wideFalseCollections() {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=kotlinx.serialization.json.Json { encodeDefaults=defaults; explicitNulls=nulls }
        val large=org.example.falseinlinedefaultslarge.models.LargeDto()
        check(large.rows?.size == 6553 && large.rows?.all { it == null } == true)
        check(large.rows !== org.example.falseinlinedefaultslarge.models.LargeDto().rows)
        val value=org.example.falseinlinedefaultswide.models.WideDto()
        check(value.p000?.isEmpty() == true && value.p239?.isEmpty() == true)
        check(codec.decodeFromString<org.example.falseinlinedefaultswide.models.WideDto>("{}").p000?.isEmpty() == true)
        val fields=codec.parseToJsonElement(codec.encodeToString(value)).jsonObject
        check(fields.size==if(defaults) 240 else 0)
        for(index in 0 until 240) {
            val field="p"+index.toString().padStart(3,'0')
            check(runCatching { codec.decodeFromString<org.example.falseinlinedefaultswide.models.WideDto>("{\"$field\":[null]}") }.isFailure)
        }
        check(codec.encodeToString(codec.decodeFromString<org.example.falseinlinedefaultswide.models.DeepDto>(VALID))==VALID)
        val model=codec.decodeFromString<org.example.falseinlinedefaultswide.models.DeepHolderDto>("{\"deep\":"+VALID+"}")
        check(codec.encodeToString(model)=="{\"deep\":"+VALID+"}")
        check(runCatching { codec.decodeFromString<org.example.falseinlinedefaultswide.models.DeepDto>(INVALID) }.isFailure)
    }
}
""".replace('INVALID',json.dumps(deep['invalid'])).replace('VALID',json.dumps(deep['valid']))
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text('package org.example.contracts\nimport kotlinx.serialization.json.jsonObject\n'+wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { wideFalseCollections(); falseInlineCollectionDefaultsContracts(); org.example.optimizedcontracts.falseInlineCollectionDefaultsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
kotlin.sourceSets.named("main") { kotlin.srcDir("build/generated-large") }
tasks.register<JavaExec>("runFalseInlineCollectionDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED FALSE INLINE COLLECTION DEFAULTS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runFalseInlineCollectionDefaultsContracts
    echo "FRESH FALSE INLINE COLLECTION DEFAULTS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh false inline collection default projects: $workspace"
