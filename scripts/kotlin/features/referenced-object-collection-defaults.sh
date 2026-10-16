#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-object-collection-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedObjectCollectionDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/referenced-object-collection-defaults.json "$project/input.json"
    cp tests/fixtures/http/referenced-object-collection-defaults.json "$project/opt.json"
    if [[ "$checked" == true ]]; then
      python3 - "$project" <<'PY'
import json
from pathlib import Path
import sys
for name in ('input.json', 'opt.json'):
    path = Path(sys.argv[1]) / name
    value = json.loads(path.read_text())
    value['components']['schemas']['AllOfRows'].update(minItems=1, maxItems=3)
    value['components']['schemas']['Holder']['properties']['inlineAllRows'].update(minItems=1, maxItems=3)
    path.write_text(json.dumps(value))
PY
    fi
    cat > "$project/src/main/kotlin/org/example/contracts/Main.kt" <<'KOTLIN'
package org.example.contracts

import kotlinx.coroutines.runBlocking

fun main() = runBlocking {
    // The optimized/locked package proves the helper binds its final relocated type.
    val optimized = org.example.refcolopt.models.HolderDto(requiredRows = listOf(org.example.refcolopt.models.ItemDto(name = "r")))
    check(optimized.inlineRows?.map { it.name } == listOf("a", "b"))
    check(optimized.namedRows == listOf(org.example.refcolopt.models.ItemDto(name = "d")))
    check(optimized.allOfRows?.single()?.name == "all" && optimized.allOfRows?.single()?.child?.note == "inner")
    check(optimized.allOfAlias?.single()?.name == "alias")
    check(optimized.allOfNested?.single()?.get("k")?.name == "internal" && optimized.allOfNested?.single()?.get("k")?.child?.note == "inner")
    check(optimized.allOfNullable?.map { it?.name } == listOf(null, "nullable") && optimized.allOfNil == null)
    check(optimized.inlineAllRows?.single()?.name == "inline" && optimized.inlineAllRows?.single()?.child?.note == "inner")
    check(optimized.inlineAllMap?.get("k")?.name == "map" && optimized.inlineAllMixed?.single()?.get("k")?.name == "internal-inline")
    check(optimized.inlineAllNullable?.map { it?.name } == listOf(null, "nullable-inline") && optimized.inlineAllNil == null)
    referencedObjectCollectionDefaultsContracts(CHECKED)
}
KOTLIN
    sed -i.bak "s/CHECKED/$checked/" "$project/src/main/kotlin/org/example/contracts/Main.kt"
    rm -f "$project/src/main/kotlin/org/example/contracts/Main.kt.bak"
    b="$project/build.gradle.kts"
    cat >> "$b" <<'GRADLE'
tasks.register<JavaExec>("runReferencedObjectCollectionDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
GRADLE
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.refcol --api-client-name RefCol --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/opt.json" --output "$project/build/generated-options" --package-name org.example.refcolopt --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED REFERENCED OBJECT COLLECTION DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedObjectCollectionDefaultsContracts
    echo "FRESH REFERENCED OBJECT COLLECTION DEFAULTS $style checked=$checked PASS: $project"
  done
done
