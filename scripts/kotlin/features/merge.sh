#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/merge-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for profile in basic short qualified; do
    project="$workspace/$style-$profile"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/MergeModelContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--merge-identical-models --validation --model-style "$style")
    if [[ "$profile" != basic ]]; then flags+=(--hoist-shared-shapes --deduplicate-models --lock-hoisted-names); fi
    if [[ "$profile" == qualified ]]; then flags+=(--qualify-hoisted-names); fi
    "$workspace/openapi-kt" generate --input tests/fixtures/http/merge-models.json --output "$project/build/generated" --package-name org.example.merge --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input tests/fixtures/http/merge-models.json --output "$project/build/generated" --package-name org.example.merge --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input tests/fixtures/http/merge-probes.json --output "$project/build/generated-limits" --package-name org.example.mergeprobes --model-name-suffix Dto --merge-identical-models --validation --model-style "$style"
    python3 - "$project" "$style" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]);plain=sys.argv[2]=='plain_class'
extra='' if plain else '''check(org.example.merge.models.AlphaDto("before").copy(value="after")==org.example.merge.models.ZetaDto("after"))
    check(runCatching { org.example.merge.models.ChildZDto(1).copy(n=0) }.isFailure)
    check(runCatching { org.example.merge.models.CheckedRecursiveZDto().copy(value=1) }.isFailure)
    check(org.example.merge.models.RecursiveZDto().copy(next=org.example.merge.models.RecursiveADto()).next != null)'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
    '''+extra+'''
    mergeModelContracts('''+str(plain).lower()+''')
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''\ntasks.register<JavaExec>("runMergeContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runMergeContracts
    printf 'FRESH MERGE PROFILE %s %s PASS: %s\n' "$style" "$profile" "$project"
  done
done
printf 'Fresh native Go CLI merge projects: %s\n' "$workspace"
