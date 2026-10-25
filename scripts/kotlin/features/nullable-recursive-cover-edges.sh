#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/nullable-recursive-cover-edges-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableRecursiveCoverEdgesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    python3 - "$project/input.json" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
d=json.loads(Path('tests/fixtures/http/nullable-recursive-cover-edges.json').read_text())
if sys.argv[2]=='on': d['components']['schemas']['UniqueHolder']['properties']['values']['uniqueItems']=True
Path(sys.argv[1]).write_text(json.dumps(d,indent=2)+'\n')
PYINPUT
    for mode in base optimized; do
      package=org.example.nullablerecursiveedges; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.nullablerecursiveedgesoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      args=(--input "$project/input.json" --output "$output" --package-name "$package" --api-client-name NullableRecursiveEdgesClient --model-name-suffix Dto --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/NullableRecursiveCoverEdgesContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'NullableRecursiveCoverEdgesContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.nullablerecursiveedges','org.example.nullablerecursiveedgesoptimized'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { nullableRecursiveCoverEdgesContracts('+checked+'); org.example.optimizedcontracts.nullableRecursiveCoverEdgesContracts('+checked+') }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runNullableRecursiveCoverEdgesContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED NULLABLE RECURSIVE COVER EDGES $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNullableRecursiveCoverEdgesContracts
    echo "FRESH NULLABLE RECURSIVE COVER EDGES PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh nullable recursive cover edge projects: $workspace"
