#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/merge-primitive-unions-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts" "$project/src/main/kotlin/org/example/optimizedcontracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/MergePrimitiveUnionContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  flags=(--model-style "$style");if [[ "$checked" == on ]];then flags+=(--validation);fi
  for mode in baseline merged optimized; do
   pkg=org.example.unionmerge;options=(--merge-identical-models)
   if [[ "$mode" == baseline ]];then pkg=org.example.unionmergebaseline;options=();fi
   if [[ "$mode" == optimized ]];then
    pkg=org.example.unionmergeoptimized;options=(--deduplicate-models --hoist-shared-shapes --lock-hoisted-names)
    "$workspace/openapi-kt" generate --input tests/fixtures/http/merge-primitive-unions.json --output "$project/build/generated/$mode" --package-name "$pkg" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
    cp "$project/build/generated/$mode/.openapi-kt-names.json" "$project/previous-lock.json"
    options+=(--merge-identical-models)
   fi
   for action in generate check; do
    check=();if [[ "$action" == check ]];then check=(--check);fi
    "$workspace/openapi-kt" generate --input tests/fixtures/http/merge-primitive-unions.json --output "$project/build/generated/$mode" --package-name "$pkg" --model-name-suffix Dto "${flags[@]}" "${options[@]}" "${check[@]}"
   done
   if [[ "$mode" == optimized ]];then cmp "$project/previous-lock.json" "$project/build/generated/$mode/.openapi-kt-names.json";fi
  done
  python3 - "$project" "$style" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);plain=sys.argv[2]=='plain_class'
consumer=(p/'src/main/kotlin/org/example/contracts/MergePrimitiveUnionContracts.kt').read_text()
(p/'src/main/kotlin/org/example/optimizedcontracts/MergePrimitiveUnionContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.unionmerge','org.example.unionmergeoptimized'))
extra='' if plain else '''check(org.example.unionmerge.models.StringZDto("old").copy(value="copy")==org.example.unionmerge.models.StringADto("copy"))
    check(org.example.unionmerge.models.IntZDto(7).component1()==7L)'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main()=runBlocking {
    val before:org.example.unionmergebaseline.models.ZDto=org.example.unionmergebaseline.models.StringZDto("wire")
    check((before as Any)!=(org.example.unionmergebaseline.models.StringADto("wire") as Any))
    check(Json.decodeFromString<org.example.unionmergebaseline.models.ZDto>("\\\"wire\\\"") == before)
    '''+extra+'''
    mergePrimitiveUnionContracts()
    org.example.optimizedcontracts.mergePrimitiveUnionContracts()
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runMergePrimitiveUnionContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]];then
   printf 'PREPARED PRIMITIVE UNION MERGE %s %s: %s\n' "$style" "$checked" "$project"
  else
   "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runMergePrimitiveUnionContracts
   printf 'FRESH PRIMITIVE UNION MERGE PROFILE %s %s PASS: %s\n' "$style" "$checked" "$project"
  fi
 done
done
printf 'Fresh primitive union merge projects: %s\n' "$workspace"
