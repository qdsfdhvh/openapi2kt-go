#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/union-field-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnionFieldDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/union-field-defaults.json "$project/input.json"
    cp tests/fixtures/http/union-field-defaults.json "$project/opt.json"
    cat > "$project/src/main/kotlin/org/example/contracts/Main.kt" <<'KOTLIN'
package org.example.contracts

import kotlinx.coroutines.runBlocking

fun main() = runBlocking {
    // The optimized/locked package proves the helper binds its final relocated type.
    val optimized = org.example.uniondefopt.models.HolderDto(requiredUnion = org.example.uniondefopt.models.HolderDto.StringRequiredUnion("r"))
    check(optimized.str is org.example.uniondefopt.models.HolderDto.StringStr)
    check(optimized.i is org.example.uniondefopt.models.HolderDto.IntI)
    val closed = optimized.closed as org.example.uniondefopt.models.ClosedDto.Variant1
    check(closed.value == "closed" && closed.age == 7 && closed.meta?.flag == true)
    check((optimized.closedNullable as org.example.uniondefopt.models.ClosedNullableDto.Variant2).value == 8 && optimized.closedNil == null)
    unionFieldDefaultsContracts(CHECKED)
}
KOTLIN
    sed -i.bak "s/CHECKED/$checked/" "$project/src/main/kotlin/org/example/contracts/Main.kt"
    rm -f "$project/src/main/kotlin/org/example/contracts/Main.kt.bak"
    b="$project/build.gradle.kts"
    cat >> "$b" <<'GRADLE'
tasks.register<JavaExec>("runUnionFieldDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
GRADLE
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.uniondef --api-client-name UnionDef --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/opt.json" --output "$project/build/generated-options" --package-name org.example.uniondefopt --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED UNION FIELD DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runUnionFieldDefaultsContracts
    echo "FRESH UNION FIELD DEFAULTS $style checked=$checked PASS: $project"
  done
done
