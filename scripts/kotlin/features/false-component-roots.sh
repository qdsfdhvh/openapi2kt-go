#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-component-roots-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseComponentRootsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.falsecomponents; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.falsecomponentsoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      suffix=; [[ "$mode" == base ]] || suffix=Dto
      args=(--input tests/fixtures/http/false-component-roots.json --output "$output" --package-name "$package" --api-client-name FalseComponentsClient --model-name-suffix "$suffix" --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/FalseComponentRootsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
import json
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts')
for name in json.loads(Path('tests/fixtures/http/false-component-roots.json').read_text())['components']['schemas']:
    optimized=optimized.replace('org.example.falsecomponents.models.'+name+' as ', 'org.example.falsecomponentsoptimized.models.'+name+'Dto as ')
(opt/'FalseComponentRootsContracts.kt').write_text(optimized)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { falseComponentRootsContracts(); org.example.optimizedcontracts.falseComponentRootsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runFalseComponentRootsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED FALSE COMPONENT ROOTS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runFalseComponentRootsContracts
    echo "FRESH FALSE COMPONENT ROOTS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh false component root projects: $workspace"
