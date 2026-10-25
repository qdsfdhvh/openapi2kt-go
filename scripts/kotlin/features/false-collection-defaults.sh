#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-collection-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseCollectionDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.falsecollectiondefaults; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.falsecollectiondefaultsoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      suffix=; [[ "$mode" == base ]] || suffix=Dto
      args=(--input tests/fixtures/http/false-collection-defaults.json --output "$output" --package-name "$package" --api-client-name FalseCollectionDefaultsClient --model-name-suffix "$suffix" --model-style "$style")
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
rows={'type':'array','items':ref('Blocked'),'default':[]}
leaf['default']=json.loads(valid)
if checked: rows['maxItems']=0
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'Blocked':False,'Rows':rows,'Wide':{'properties':{f'p{i:03d}':ref('Rows') for i in range(240)}},'Deep':leaf,'DeepHolder':{'required':['deep'],'properties':{'deep':ref('Deep')}},'MaybeBlocked':{'$ref':'#/components/schemas/Blocked','nullable':True},'Large':{'type':'array','items':ref('MaybeBlocked'),'default':[None]*6553}}}}))
(p/'deep-values.json').write_text(json.dumps({'valid':valid,'invalid':invalid}))
PYWIDE
    wide_args=(--input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.falsecollectiondefaultswide --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}"
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/FalseCollectionDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
import json
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts')
optimized=optimized.replace('org.example.falsecollectiondefaults.FalseCollectionDefaultsClient','org.example.falsecollectiondefaultsoptimized.FalseCollectionDefaultsClient')
for name in json.loads(Path('tests/fixtures/http/false-collection-defaults.json').read_text())['components']['schemas']:
    optimized=optimized.replace('org.example.falsecollectiondefaults.models.'+name+' as ', 'org.example.falsecollectiondefaultsoptimized.models.'+name+'Dto as ')
    optimized=optimized.replace('org.example.falsecollectiondefaults.models.`'+name+' default` as ', 'org.example.falsecollectiondefaultsoptimized.models.`'+name+'Dto default` as ')
(opt/'FalseCollectionDefaultsContracts.kt').write_text(optimized)
deep=json.loads((p/'deep-values.json').read_text())
wide=r"""
fun wideFalseCollections() {
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val codec=kotlinx.serialization.json.Json { encodeDefaults=defaults; explicitNulls=nulls }
        val value=codec.decodeFromString<org.example.falsecollectiondefaultswide.models.WideDto>("{}")
        check(org.example.falsecollectiondefaultswide.models.`RowsDto default`().isEmpty())
        check(codec.encodeToString(org.example.falsecollectiondefaultswide.models.`DeepDto default`())==VALID)
        val first=org.example.falsecollectiondefaultswide.models.`LargeDto default`()
        val second=org.example.falsecollectiondefaultswide.models.`LargeDto default`()
        check(first.size==6553 && second.size==6553 && first !== second)
        (first as MutableList<org.example.falsecollectiondefaultswide.models.BlockedDto?>).clear()
        check(second.size==6553)

        val fields=codec.parseToJsonElement(codec.encodeToString(value)).jsonObject
        check(fields.size==if(defaults && nulls) 240 else 0)
        for(index in 0 until 240) {
            val field="p"+index.toString().padStart(3,'0')
            check(runCatching { codec.decodeFromString<org.example.falsecollectiondefaultswide.models.WideDto>("{\"$field\":[null]}") }.isFailure)
        }
        check(codec.encodeToString(codec.decodeFromString<org.example.falsecollectiondefaultswide.models.DeepDto>(VALID))==VALID)
        val model=codec.decodeFromString<org.example.falsecollectiondefaultswide.models.DeepHolderDto>("{\"deep\":"+VALID+"}")
        check(codec.encodeToString(model)=="{\"deep\":"+VALID+"}")
        check(runCatching { codec.decodeFromString<org.example.falsecollectiondefaultswide.models.DeepDto>(INVALID) }.isFailure)
    }
}
""".replace('INVALID',json.dumps(deep['invalid'])).replace('VALID',json.dumps(deep['valid']))
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text('package org.example.contracts\nimport kotlinx.serialization.json.jsonObject\n'+wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { wideFalseCollections(); falseCollectionDefaultsContracts(); org.example.optimizedcontracts.falseCollectionDefaultsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runFalseCollectionDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED FALSE COLLECTION DEFAULTS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runFalseCollectionDefaultsContracts
    echo "FRESH FALSE COLLECTION DEFAULTS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh false collection default projects: $workspace"
