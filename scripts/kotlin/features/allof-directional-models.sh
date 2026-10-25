#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/allof-directional-models-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/AllOfDirectionalModelContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json, sys
project = Path(sys.argv[1])
document = json.loads(Path('tests/fixtures/http/allof-directional-models.json').read_text())
if sys.argv[2] == 'on':
    document['components']['schemas']['Account']['allOf'][1]['properties']['name']['minLength'] = 1
(project/'input.json').write_text(json.dumps(document, indent=2)+'\n')
wide=json.loads(json.dumps(document))
wide['components']['schemas']['Account']={'allOf':[{'properties':{'id':{'type':'string','readOnly':True}},'required':['id']},{'properties':dict({'password':{'type':'string','writeOnly':True}},**{'p'+str(i):{'type':'string'} for i in range(238)}),'required':['password']}]}
(project/'wide.json').write_text(json.dumps(wide,indent=2)+'\n')

PYINPUT
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    for profile in allofdirectional allofdirectionaloptions; do
      target="$project/build/generated"
      options=()
      if [[ "$profile" == allofdirectionaloptions ]]; then
        target="$project/build/generated-options"
        options=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$target" --package-name "org.example.$profile" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$target" --package-name "org.example.$profile" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
    done
    "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-limits" --package-name org.example.allofdirectionalwide --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-limits" --package-name org.example.allofdirectionalwide --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" "$checked" <<'PY'
from pathlib import Path
import sys
project = Path(sys.argv[1])
checked = str(sys.argv[2] == 'on').lower()
(project/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { check(org.example.allofdirectionalwide.api.models.SaveAccountRequestDto(password="wide").p237==null); allOfDirectionalModelContracts('''+checked+''') }
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
