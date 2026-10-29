#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-hierarchy-references-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleHierarchyReferencesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/tuple-hierarchy-references.json').read_text())
# An independent 240-field hierarchy tuple DTO exercises constructor and classifier width.
d['components']['schemas']['Wide']={'properties':{f'v{i:03d}':{'type':'array','prefixItems':[{'$ref':'#/components/schemas/First'}]} for i in range(240)}}
if checked: d['components']['schemas']['Holder']['properties']['required']['minItems']=1
(p/'input.json').write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
consumer=p/'src/main/kotlin/org/example/contracts/TupleHierarchyReferencesContracts.kt'
(p/'src/main/kotlin/org/example/contracts/OptimizedTupleHierarchyReferencesContracts.kt').write_text(consumer.read_text().replace('org.example.hierarchytuples','org.example.hierarchytuplesoptimized').replace('HierarchyTuple','OptimizedHierarchyTuple').replace('hierarchyTuple','optimizedHierarchyTuple').replace('tupleHierarchyReferences','optimizedTupleHierarchyReferences'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
    tupleHierarchyReferencesContracts('''+str(checked).lower()+''')
    optimizedTupleHierarchyReferencesContracts('''+str(checked).lower()+''')
}
''')
build=p/'build.gradle.kts'
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runTupleHierarchyReferencesContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
    for mode in baseline optimized; do
      output="$project/build/generated"
      package=org.example.hierarchytuples
      options=()
      if [[ "$mode" == optimized ]]; then
        output="$project/build/generated-limits"
        package=org.example.hierarchytuplesoptimized
        options=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
    done
    python3 - "$project/build/generated-limits/.openapi-kt-names.json" <<'PYLOCK'
import json,sys
assert json.load(open(sys.argv[1]))['entries'], 'optimized tuple profile requires actual locked hoist allocations'
PYLOCK
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleHierarchyReferencesContracts
    echo "FRESH HIERARCHY TUPLES PROFILE $style $checked PASS: $project"
  done
done
