#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-root-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleRootDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.tupleroot; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.tuplerootoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      args=(--input tests/fixtures/http/tuple-root-defaults.json --output "$output" --package-name "$package" --api-client-name TupleRootClient --model-name-suffix Dto --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    python3 - "$project" "$checked" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1])
wide={'properties':{'p'+str(i):{'type':'array','prefixItems':[{'type':'integer'}]} for i in range(240)}}
schemas={'Wide':wide}
if sys.argv[2]=='on':
 schemas['Counted']={'type':'array','prefixItems':[{'type':'integer'}],'items':{'type':'integer'},'minItems':1,'maxItems':3,'uniqueItems':True,'default':[1,2]}
 schemas['CheckedHolder']={'properties':{'rows':{'$ref':'#/components/schemas/Counted'}}}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}})+'\n')
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-options/limits" --package-name org.example.tuplerootlimits --model-name-suffix Dto --model-style "$style" "${validation[@]}"
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-options/limits" --package-name org.example.tuplerootlimits --model-name-suffix Dto --model-style "$style" "${validation[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/TupleRootDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'TupleRootDefaultsContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.tupleroot','org.example.tuplerootoptimized'))
extra="check(org.example.tuplerootlimits.models.WideDto().p239 == null); "
if checked=='true':
 extra+='check(org.example.tuplerootlimits.models.`CountedDto default`() == listOf(1,2)); check(runCatching { org.example.tuplerootlimits.models.CheckedHolderDto(rows=emptyList()) }.isFailure); check(runCatching { org.example.tuplerootlimits.models.CheckedHolderDto(rows=listOf(1,1)) }.isFailure); '
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { '+extra+' tupleRootDefaultsContracts(); org.example.optimizedcontracts.tupleRootDefaultsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runTupleRootDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED TUPLE ROOT DEFAULTS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleRootDefaultsContracts
    echo "FRESH TUPLE ROOT DEFAULTS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh tuple root default projects: $workspace"
