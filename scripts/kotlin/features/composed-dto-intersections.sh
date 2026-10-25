#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/composed-dto-intersections-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ComposedDTOIntersectionsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys,json
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
d=json.loads(Path('tests/fixtures/http/composed-dto-intersections.json').read_text())
if checked=='true':d['components']['schemas']['Owner']['allOf'][0]['properties']['payload']['properties']['node']['properties']['count']['minimum']=1
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
nullable=json.loads(Path('tests/fixtures/http/nullable-composed-dto-intersections.json').read_text())
if checked=='true':nullable['components']['schemas']['Owner']['allOf'][0]['properties']['payload']['properties']['node']['properties']['count']['minimum']=1
(p/'nullable.json').write_text(json.dumps(nullable,indent=2)+'\n')
source=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableComposedDTOIntersectionsContracts.kt').read_text()
(p/'src/main/kotlin/org/example/contracts/NullableComposedDTOIntersectionsContracts.kt').write_text(source)
(opt_source:=p/'src/main/kotlin/org/example/optimizedcontracts').mkdir(parents=True,exist_ok=True)
(opt_source/'NullableComposedDTOIntersectionsContracts.kt').write_text(source.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.composeddtonullable','org.example.composeddtooptimizednullable'))

wide={'openapi':'3.1.0','components':{'schemas':{'Owner':{'allOf':[{'properties':{'node':{'properties':{'p'+str(i):{'type':'integer'} for i in range(239)}}}},{'properties':{'ownerExtra':{'type':'string'}}}]},'Hops':{'properties':{'v':{'$ref':'#/components/schemas/Owner/allOf/0/properties/node'}}},'Bridge':{'$ref':'#/components/schemas/Hops/properties/v'},'Consumer':{'allOf':[{'$ref':'#/components/schemas/Bridge'},{'properties':{'extra':{'type':'string'}}}]}}}}
(p/'wide.json').write_text(json.dumps(wide)+'\n')


consumer=(p/'src/main/kotlin/org/example/contracts/ComposedDTOIntersectionsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'ComposedDTOIntersectionsContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.composeddto','org.example.composeddtooptimized'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { check(org.example.composeddtowide.models.ConsumerDto(p238=238,extra="wide").p238==238); composedDTOIntersectionsContracts('+checked+'); nullableComposedDTOIntersectionsContracts('+checked+'); org.example.optimizedcontracts.composedDTOIntersectionsContracts('+checked+'); org.example.optimizedcontracts.nullableComposedDTOIntersectionsContracts('+checked+') }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
kotlin.sourceSets.getByName("main").kotlin.srcDirs("build/generated-nullable", "build/generated-limits-nullable")
tasks.register<JavaExec>("runComposedDTOIntersectionsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    for mode in base optimized wide; do
      package=org.example.composeddto; output="$project/build/generated"; options=(); input="$project/input.json"
      if [[ "$mode" == optimized ]]; then
        package=org.example.composeddtooptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      elif [[ "$mode" == wide ]]; then
        package=org.example.composeddtowide;output="$project/build/generated-options";input="$project/wide.json"
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
      echo "PREPARED COMPOSED DTO INTERSECTIONS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runComposedDTOIntersectionsContracts
    echo "FRESH COMPOSED DTO INTERSECTIONS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh bridge intersection projects: $workspace"
