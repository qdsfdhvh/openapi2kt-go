#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/root-dictionary-allof-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RootDictionaryAllOfWrapperContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    python3 - "$project/input.json" "$checked" <<'PYINPUT'
from pathlib import Path
import json, sys

doc = json.loads(Path("tests/fixtures/http/root-dictionary-allof-wrappers.json").read_text())
if sys.argv[2] == "on":
    target = doc["components"]["schemas"]["ZMap"]
    target.update(minProperties=1, maxProperties=2)
    target["additionalProperties"]["minLength"] = 2
Path(sys.argv[1]).write_text(json.dumps(doc, indent=2) + "\n")
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" \
      --package-name org.example.dictionarywrapper --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" \
      --package-name org.example.dictionarywrapper --model-name-suffix Dto "${flags[@]}" --check
    if [[ "$checked" == on ]]; then check_literal=true; else check_literal=false; fi
    cat > "$project/src/main/kotlin/org/example/contracts/Main.kt" <<PYMAIN
package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking { rootDictionaryAllOfWrapperContracts($check_literal) }
PYMAIN
    cat >> "$project/build.gradle.kts" <<'PYGRADLE'
tasks.register<JavaExec>("runRootDictionaryAllOfWrapperContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
PYGRADLE
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runRootDictionaryAllOfWrapperContracts
    echo "FRESH ROOT DICTIONARY ALLOF WRAPPERS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh root dictionary allOf wrapper projects: $workspace"
