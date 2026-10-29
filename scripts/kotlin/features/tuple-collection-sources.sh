#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-collection-sources-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleCollectionSourcesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/tuple-collection-sources.json').read_text())
if checked:
    def constrain(v):
        if isinstance(v,dict):
            if v.get('type')=='integer':v['minimum']=0
            for child in v.values():constrain(child)
        elif isinstance(v,list):
            for child in v:constrain(child)
    constrain(d['components']);constrain(d['paths']['/patch'])
    d['components']['schemas']['Holder']['properties']['required']['prefixItems'][0]['minItems']=1
# Independent 240 outer fields preserve source capture and final helper types.
d['components']['schemas']['Wide']={'properties':{f'v{i:03d}':{'type':'array','prefixItems':[{'type':'array','items':{'type':'integer'}}],'default':[[1]]} for i in range(240)}}
(p/'input.json').write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
consumer=p/'src/main/kotlin/org/example/contracts/TupleCollectionSourcesContracts.kt'
(p/'src/main/kotlin/org/example/contracts/OptimizedTupleCollectionSourcesContracts.kt').write_text(consumer.read_text().replace('org.example.collectiontuples','org.example.collectiontuplesoptimized').replace('CollectionTuple','OptimizedCollectionTuple').replace('collectionTuple','optimizedCollectionTuple').replace('tupleCollectionSources','optimizedTupleCollectionSources'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
    tupleCollectionSourcesContracts('''+str(checked).lower()+''')
    optimizedTupleCollectionSourcesContracts('''+str(checked).lower()+''')
}
''')
build=p/'build.gradle.kts'
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runTupleCollectionSourcesContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
    for mode in baseline optimized; do
      output="$project/build/generated"
      package=org.example.collectiontuples
      options=()
      if [[ "$mode" == optimized ]]; then
        output="$project/build/generated-limits"
        package=org.example.collectiontuplesoptimized
        options=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
    done
    python3 - "$project/build/generated-limits/.openapi-kt-names.json" <<'PYLOCK'
import json,sys
assert json.load(open(sys.argv[1]))['entries'], 'optimized tuple profile requires actual locked hoist allocations'
PYLOCK
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleCollectionSourcesContracts
    echo "FRESH COLLECTION TUPLE PROFILE $style $checked PASS: $project"
  done
done
