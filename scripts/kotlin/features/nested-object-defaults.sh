#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/nested-object-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedObjectDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/nested-object-defaults.json "$project/input.json"
    cp tests/fixtures/http/nested-object-defaults.json "$project/opt.json"
    cat > "$project/src/main/kotlin/org/example/contracts/Main.kt" <<'KOTLIN'
package org.example.contracts

import kotlinx.coroutines.runBlocking

fun main() = runBlocking {
    // The optimized/locked package proves the helper binds its final relocated type.
    val optimized = org.example.nestedobjectopt.models.HolderDto(requiredCfg = org.example.nestedobjectopt.models.HolderDto.RequiredCfg(id = "x"))
    check(optimized.cfg?.name == "outer" && optimized.cfg?.age == 9 && optimized.cfg?.child?.flag == false)
    check(optimized.cfg?.tags == listOf("a", "b") && optimized.cfg?.counts == mapOf("k" to 1))
    check(optimized.allOfCfg?.z == "merged" && optimized.allOfCfg?.count == 5 && optimized.allOfCfg?.meta?.flag == true)
    check(optimized.allOfNull == null)
    check(optimized.cfg?.composed?.label == "outer member" && optimized.cfg?.composed?.count == 3 && optimized.cfg?.composed?.child?.flag == false)
    check(optimized.cfg?.linked?.label == "outer linked" && optimized.cfg?.linked?.count == 2 && optimized.cfg?.linked?.child?.flag == true)
    check(optimized.cfg?.viaAlias?.label == "outer alias" && optimized.cfg?.viaAlias?.count == 2 && optimized.cfg?.viaAlias?.child?.flag == true)
    check(optimized.cfg?.aliasNull == null)
    check(optimized.cfg?.owned?.label == "outer owned" && optimized.cfg?.owned?.child?.flag == true && optimized.cfg?.ownedAlias == null)
    check(optimized.cfg?.ownedRow?.label == "outer row" && optimized.cfg?.ownedVariant?.label == "outer variant")
    nestedObjectDefaultsContracts(CHECKED)
}
KOTLIN
    sed -i.bak "s/CHECKED/$checked/" "$project/src/main/kotlin/org/example/contracts/Main.kt"
    rm -f "$project/src/main/kotlin/org/example/contracts/Main.kt.bak"
    b="$project/build.gradle.kts"
    cat >> "$b" <<'GRADLE'
tasks.register<JavaExec>("runNestedObjectDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
GRADLE
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.nestedobject --api-client-name NestedObject --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/opt.json" --output "$project/build/generated-options" --package-name org.example.nestedobjectopt --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED NESTED OBJECT DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNestedObjectDefaultsContracts
    echo "FRESH NESTED OBJECT DEFAULTS $style checked=$checked PASS: $project"
  done
done
