#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-structured-const-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleStructuredConstContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/tuple-structured-const.json').read_text())
if checked:
    d['components']['schemas']['Holder']['properties']['required']['minItems']=1
# Independent source nodes and actual const codec default helpers at the JVM slot cap.
d['components']['schemas']['Wide']={'properties':{f'v{i:03d}':{'type':'array','prefixItems':[{'const':{'fixed':1}}],'default':[{'fixed':1}]} for i in range(240)}}
d['components']['schemas']['Holder']['properties']['nestedDefault']={'type':'array','prefixItems':[{'type':'array','items':{'const':{'fixed':1}}}],'default':[[{'fixed':1}]]}
(p/'input.json').write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
consumer=p/'src/main/kotlin/org/example/contracts/TupleStructuredConstContracts.kt'
(p/'src/main/kotlin/org/example/contracts/OptimizedTupleStructuredConstContracts.kt').write_text(consumer.read_text().replace('org.example.structuredtuples','org.example.structuredtuplesoptimized').replace('StructuredTuple','OptimizedStructuredTuple').replace('structuredTuple','optimizedStructuredTuple').replace('tupleStructuredConst','optimizedTupleStructuredConst'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
fun main() = runBlocking {
    tupleStructuredConstContracts('''+str(checked).lower()+''')
    optimizedTupleStructuredConstContracts('''+str(checked).lower()+''')
    val fixed = JsonObject(mapOf("fixed" to JsonPrimitive(1)))
    val plain = org.example.structuredtuples.models.HolderDto(required = listOf(fixed))
    val optimized = org.example.structuredtuplesoptimized.models.HolderDto(required = listOf(fixed))
    check(plain.nestedDefault == listOf(listOf(fixed)))
    check(optimized.nestedDefault == listOf(listOf(fixed)))
    check(plain.nestedDefault !== org.example.structuredtuples.models.HolderDto(required = listOf(fixed)).nestedDefault)
}
''')
build=p/'build.gradle.kts'
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runTupleStructuredConstContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
    for mode in baseline optimized; do
      output="$project/build/generated"
      package=org.example.structuredtuples
      options=()
      if [[ "$mode" == optimized ]]; then
        output="$project/build/generated-limits"
        package=org.example.structuredtuplesoptimized
        options=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
    done
    python3 - "$project/build/generated-limits/.openapi-kt-names.json" <<'PYLOCK'
import json,sys
assert json.load(open(sys.argv[1]))['entries'], 'optimized tuple profile requires actual locked hoist allocations'
PYLOCK
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleStructuredConstContracts
    echo "FRESH STRUCTURED TUPLE PROFILE $style $checked PASS: $project"
  done
done
