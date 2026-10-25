#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-alias-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseAliasDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.falsealiasdefaults; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.falsealiasdefaultsoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      suffix=; [[ "$mode" == base ]] || suffix=Dto
      args=(--input tests/fixtures/http/false-alias-defaults.json --output "$output" --package-name "$package" --api-client-name FalseAliasDefaultsClient --model-name-suffix "$suffix" --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    python3 - "$project" <<'PYWIDE'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);ref=lambda name,**extra:{'$ref':'#/components/schemas/'+name,**extra}
schemas={'Blocked':False,'Maybe':ref('Blocked',nullable=True,default=None),'Wide':{'properties':{f'p{i:03d}':ref('Maybe') for i in range(240)}}}
previous='Blocked'
for index in range(48):
    name=f'A{index:02d}';schemas[name]=ref(previous,nullable=True,default=None);previous=name
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
PYWIDE
    wide_args=(--input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.falsealiasdefaultswide --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}"
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/FalseAliasDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
import json
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts')
optimized=optimized.replace('org.example.falsealiasdefaults.FalseAliasDefaultsClient','org.example.falsealiasdefaultsoptimized.FalseAliasDefaultsClient')
for name in json.loads(Path('tests/fixtures/http/false-alias-defaults.json').read_text())['components']['schemas']:
    optimized=optimized.replace('org.example.falsealiasdefaults.models.'+name+' as ', 'org.example.falsealiasdefaultsoptimized.models.'+name+'Dto as ')
    optimized=optimized.replace('org.example.falsealiasdefaults.models.`'+name+' default` as ', 'org.example.falsealiasdefaultsoptimized.models.`'+name+'Dto default` as ')
(opt/'FalseAliasDefaultsContracts.kt').write_text(optimized)
wide=r"""
fun wideFalseAliases() {
    check(org.example.falsealiasdefaultswide.models.`A47Dto default`() == null)
    check(org.example.falsealiasdefaultswide.models.`MaybeDto default`() == null)
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec=kotlinx.serialization.json.Json { encodeDefaults=defaults; explicitNulls=nulls }
        val value=codec.decodeFromString<org.example.falsealiasdefaultswide.models.WideDto>("{}")
        val fields=codec.parseToJsonElement(codec.encodeToString(value)).jsonObject
        check(fields.size == if(defaults && nulls) 240 else 0)
        for(index in 0 until 240) {
            val field="p"+index.toString().padStart(3,'0')
            check(runCatching { codec.decodeFromString<org.example.falsealiasdefaultswide.models.WideDto>("{\"$field\":true}") }.isFailure)
        }
    }
}
"""
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text('package org.example.contracts\nimport kotlinx.serialization.json.jsonObject\n'+wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { wideFalseAliases(); falseAliasDefaultsContracts(); org.example.optimizedcontracts.falseAliasDefaultsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runFalseAliasDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED FALSE ALIAS DEFAULTS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runFalseAliasDefaultsContracts
    echo "FRESH FALSE ALIAS DEFAULTS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh false alias default projects: $workspace"
