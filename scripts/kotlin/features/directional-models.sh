#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/directional-models-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DirectionalModelContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json, sys
project = Path(sys.argv[1])
document = json.loads(Path('tests/fixtures/http/directional-models.json').read_text())
if sys.argv[2] == 'on':
    document['components']['schemas']['Account']['properties']['name']['minLength'] = 1
(project/'input.json').write_text(json.dumps(document, indent=2)+'\n')
PYINPUT
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    for profile in directional directionaloptions; do
      target="$project/build/generated"
      options=()
      if [[ "$profile" == directionaloptions ]]; then
        target="$project/build/generated-options"
        options=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$target" --package-name "org.example.$profile" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$target" --package-name "org.example.$profile" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
    done
    python3 - "$project" "$checked" <<'PY'
from pathlib import Path
import sys
project = Path(sys.argv[1])
checked = str(sys.argv[2] == 'on').lower()
(project/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { directionalModelContracts('''+checked+''') }
''')
build = project/'build.gradle.kts'
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runDirectionalContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runDirectionalContracts
    echo "FRESH DIRECTIONAL DTO PROFILE $style $checked PASS: $project"
  done
done
