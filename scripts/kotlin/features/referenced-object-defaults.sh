#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-object-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedObjectDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/referenced-object-defaults.json "$project/input.json"
    cp tests/fixtures/http/referenced-object-defaults.json "$project/opt.json"
    cat > "$project/src/main/kotlin/org/example/contracts/Main.kt" <<'KOTLIN'
package org.example.contracts

import kotlinx.coroutines.runBlocking

fun main() = runBlocking {
    // The optimized/locked package proves the helper binds its final relocated type.
    val optimized = org.example.refobjectopt.models.HolderDto(requiredCfg = org.example.refobjectopt.models.InnerDto(name = "required"))
    check(optimized.cfg?.name == "outer" && optimized.cfg?.age == 9 && optimized.cfg?.child?.flag == false)
    check(optimized.aliasCfg?.name == "alias" && optimized.nilCfg == null)
    check(optimized.allOfCfg?.name == "composed" && optimized.allOfCfg?.n == 2 && optimized.allOfCfg?.child?.flag == true)
    check(optimized.aliasAllOfCfg?.name == "alias-composed" && optimized.aliasAllOfCfg?.n == 3 && optimized.allOfNil == null)
    check(optimized.bridgedAllOfCfg?.name == "bridge-composed" && optimized.bridgedAllOfCfg?.n == 4 && optimized.bridgedAllOfCfg?.child?.flag == true)
    check(optimized.internalCfg?.name == "internal" && optimized.internalCfg?.n == 2 && optimized.internalCfg?.child?.flag == true)
    check(optimized.internalAliasCfg?.name == "internal-alias" && optimized.internalBridgeCfg?.name == "internal-bridge")
    check(optimized.internalListCfg?.label == "list" && optimized.internalMapCfg?.label == "map")
    check(optimized.internalRootListCfg?.label == "root-list" && optimized.internalRootMapCfg?.label == "root-map")
    check(optimized.internalVariantCfg?.label == "variant" && optimized.internalVariantCfg?.kind == "all" && optimized.internalVariantCfg?.n == 7)
    check(optimized.internalMaybe?.label == "nullable" && optimized.internalNull == null)
    check(org.example.refobjectopt.models.HolderDto(requiredCfg = org.example.refobjectopt.models.InnerDto(name = "required")).allOfCfg !== optimized.allOfCfg)
    referencedObjectDefaultsContracts(CHECKED)
}
KOTLIN
    sed -i.bak "s/CHECKED/$checked/" "$project/src/main/kotlin/org/example/contracts/Main.kt"
    rm -f "$project/src/main/kotlin/org/example/contracts/Main.kt.bak"
    b="$project/build.gradle.kts"
    cat >> "$b" <<'GRADLE'
tasks.register<JavaExec>("runReferencedObjectDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
GRADLE
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.refobject --api-client-name RefObject --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/opt.json" --output "$project/build/generated-options" --package-name org.example.refobjectopt --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED REFERENCED OBJECT DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedObjectDefaultsContracts
    echo "FRESH REFERENCED OBJECT DEFAULTS $style checked=$checked PASS: $project"
  done
done
