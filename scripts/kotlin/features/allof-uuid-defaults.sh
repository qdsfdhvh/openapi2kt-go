#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/allof-uuid-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/AllOfUUIDDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.allofuuid; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.allofuuidoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      args=(--input tests/fixtures/http/allof-uuid-defaults.json --output "$output" --package-name "$package" --api-client-name UUIDDefaultsClient --model-name-suffix Dto --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/AllOfUUIDDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'AllOfUUIDDefaultsContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.allofuuid','org.example.allofuuidoptimized'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { allOfUUIDDefaultsContracts(); org.example.optimizedcontracts.allOfUUIDDefaultsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runAllOfUUIDDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED ALLOF UUID DEFAULTS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runAllOfUUIDDefaultsContracts
    echo "FRESH ALLOF UUID DEFAULTS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh allOf UUID default projects: $workspace"
