#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-reference-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleReferenceDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/tuple-reference-defaults.json').read_text())
if checked:
    for name,values in [('Integer',[1,2]),('Long',[9007199254740993]),('Float',[0.1]),('Double',[2.5]),('Flag',[True])]:
        d['components']['schemas'][name]['enum']=values
    d['components']['schemas']['Item']['minProperties']=1
    d['components']['schemas']['Composed']['minProperties']=1
# A separate 240-field DTO compiles actual referenced elements/default helpers.
d['components']['schemas']['Wide']={'properties':{f'v{i:03d}':{'type':'array','prefixItems':[{'$ref':'#/components/schemas/Integer'}],'default':[1]} for i in range(240)}}
(p/'input.json').write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
consumer=p/'src/main/kotlin/org/example/contracts/TupleReferenceDefaultsContracts.kt'
(p/'src/main/kotlin/org/example/contracts/OptimizedTupleReferenceDefaultsContracts.kt').write_text(consumer.read_text().replace('org.example.reftupledefaults','org.example.reftupledefaultsoptimized').replace('RefDefaultTuple','OptimizedRefDefaultTuple').replace('refDefaultTuple','optimizedRefDefaultTuple').replace('tupleReferenceDefaults','optimizedTupleReferenceDefaults'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
    tupleReferenceDefaultsContracts('''+str(checked).lower()+''')
    optimizedTupleReferenceDefaultsContracts('''+str(checked).lower()+''')
}
''')
build=p/'build.gradle.kts'
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runTupleReferenceDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
    for mode in baseline optimized; do
      output="$project/build/generated"
      package=org.example.reftupledefaults
      options=()
      if [[ "$mode" == optimized ]]; then
        output="$project/build/generated-limits"
        package=org.example.reftupledefaultsoptimized
        options=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
    done
    python3 - "$project/build/generated-limits/.openapi-kt-names.json" <<'PYLOCK'
import json,sys
assert json.load(open(sys.argv[1]))['entries'], 'optimized tuple profile requires actual locked hoist allocations'
PYLOCK
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleReferenceDefaultsContracts
    echo "FRESH TUPLE REFERENCE DEFAULTS PROFILE $style $checked PASS: $project"
  done
done
