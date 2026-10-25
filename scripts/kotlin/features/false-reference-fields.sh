#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-reference-fields-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseReferenceFieldsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.falsefields; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.falsefieldsoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      suffix=; [[ "$mode" == base ]] || suffix=Dto
      args=(--input tests/fixtures/http/false-reference-fields.json --output "$output" --package-name "$package" --api-client-name FalseFieldsClient --model-name-suffix "$suffix" --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    python3 - "$project" <<'PYWIDE'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);props={f'p{i:03d}':{'$ref':'#/components/schemas/Blocked'} for i in range(240)}
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'Blocked':False,'Wide':{'properties':props}}}}))
PYWIDE
    wide_args=(--input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.falsefieldswide --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}"
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/FalseReferenceFieldsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
import json
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts')
for name in json.loads(Path('tests/fixtures/http/false-reference-fields.json').read_text())['components']['schemas']:
    optimized=optimized.replace('org.example.falsefields.models.'+name+' as ', 'org.example.falsefieldsoptimized.models.'+name+'Dto as ')
(opt/'FalseReferenceFieldsContracts.kt').write_text(optimized)
wide = r"""
fun wideFalseFields() {
    for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
        val json = kotlinx.serialization.json.Json { encodeDefaults = defaults; explicitNulls = nulls }
        val value = json.decodeFromString<org.example.falsefieldswide.models.WideDto>("{}")
        check(json.encodeToString(value) == "{}")
        for (index in 0 until 240) {
            val field = "p" + index.toString().padStart(3, '0')
            check(runCatching { json.decodeFromString<org.example.falsefieldswide.models.WideDto>("{\"$field\":null}") }.isFailure)
        }
    }
}
"""
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text('package org.example.contracts\n'+wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { wideFalseFields(); falseReferenceFieldsContracts(); org.example.optimizedcontracts.falseReferenceFieldsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runFalseReferenceFieldsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED FALSE REFERENCE FIELDS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runFalseReferenceFieldsContracts
    echo "FRESH FALSE REFERENCE FIELDS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh false reference field projects: $workspace"
