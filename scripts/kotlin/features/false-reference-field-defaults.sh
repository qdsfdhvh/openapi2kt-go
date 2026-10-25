#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-reference-field-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseReferenceFieldDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.falsefielddefaults; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.falsefielddefaultsoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      suffix=; [[ "$mode" == base ]] || suffix=Dto
      args=(--input tests/fixtures/http/false-reference-field-defaults.json --output "$output" --package-name "$package" --api-client-name FalseFieldDefaultsClient --model-name-suffix "$suffix" --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    python3 - "$project" <<'PYWIDE'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);ref=lambda name,**extra:{'$ref':'#/components/schemas/'+name,**extra}
properties={f'p{i:03d}':ref('Maybe',default=None) for i in range(240)}
schemas={'Blocked':False,'Maybe':ref('Blocked',nullable=True,default=None),'WideOptional':{'properties':properties},'WideRequired':{'required':list(properties),'properties':properties}}
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
PYWIDE
    wide_args=(--input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.falsefielddefaultswide --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}"
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/FalseReferenceFieldDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
import json
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts')
optimized=optimized.replace('org.example.falsefielddefaults.FalseFieldDefaultsClient','org.example.falsefielddefaultsoptimized.FalseFieldDefaultsClient')
for name in json.loads(Path('tests/fixtures/http/false-reference-field-defaults.json').read_text())['components']['schemas']:
    optimized=optimized.replace('org.example.falsefielddefaults.models.'+name+' as ', 'org.example.falsefielddefaultsoptimized.models.'+name+'Dto as ')
    optimized=optimized.replace('org.example.falsefielddefaults.models.`'+name+' default` as ', 'org.example.falsefielddefaultsoptimized.models.`'+name+'Dto default` as ')
(opt/'FalseReferenceFieldDefaultsContracts.kt').write_text(optimized)
arguments=','.join(['null']*240)
wide=r"""
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
fun wideFalseFieldDefaults() {
    val required=org.example.falsefielddefaultswide.models.WideRequiredDto(ARGUMENTS)
    val descriptor=kotlinx.serialization.serializer<org.example.falsefielddefaultswide.models.WideRequiredDto>().descriptor
    for(index in 0 until 240) check(!descriptor.isElementOptional(index))
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=kotlinx.serialization.json.Json { encodeDefaults=defaults; explicitNulls=nulls }
        val optional=codec.decodeFromString<org.example.falsefielddefaultswide.models.WideOptionalDto>("{}")
        val fields=codec.parseToJsonElement(codec.encodeToString(optional)).jsonObject
        check(fields.size == if(defaults && nulls) 240 else 0)
        check(codec.parseToJsonElement(codec.encodeToString(required)).jsonObject.size == if(nulls) 240 else 0)
        val missing=runCatching { codec.decodeFromString<org.example.falsefielddefaultswide.models.WideRequiredDto>("{}") }
        if(nulls) check(missing.isFailure) else check(missing.isSuccess)
        for(index in 0 until 240) {
            val field="p"+index.toString().padStart(3,'0')
            check(runCatching { codec.decodeFromString<org.example.falsefielddefaultswide.models.WideOptionalDto>("{\"$field\":true}") }.isFailure)
        }
    }
}
""".replace('ARGUMENTS',arguments)
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text('package org.example.contracts\nimport kotlinx.serialization.json.jsonObject\n'+wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { wideFalseFieldDefaults(); falseReferenceFieldDefaultsContracts(); org.example.optimizedcontracts.falseReferenceFieldDefaultsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runFalseReferenceFieldDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED FALSE REFERENCE FIELD DEFAULTS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runFalseReferenceFieldDefaultsContracts
    echo "FRESH FALSE REFERENCE FIELD DEFAULTS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh false reference field default projects: $workspace"
