#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/parameter-styles-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ParameterStylesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$checked" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/parameter-styles.json').read_text())
if checked:
    d['paths']['/deepStrings']['get']['parameters'][0]['schema']['additionalProperties']['minLength']=2
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { parameterStylesContracts('''+str(checked).lower()+''') }
''')
PY
  flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.parameterstyles --api-client-name ParameterStyles --model-name-suffix Dto --model-style "$style")
  if [[ "$checked" == on ]]; then flags+=(--validation); fi
  if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names); fi
  "$workspace/openapi-kt" generate "${flags[@]}"
  "$workspace/openapi-kt" generate "${flags[@]}" --check
  cat >> "$project/build.gradle.kts" <<'GRADLE'
tasks.register<JavaExec>("runParameterStylesContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
GRADLE
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
    echo "PREPARED PARAMETER STYLES $style $checked: $project"
    continue
  fi
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runParameterStylesContracts
 done
done
echo "Fresh parameter style projects: $workspace"
