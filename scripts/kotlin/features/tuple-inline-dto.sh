#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-inline-dto-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleInlineDtoContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/tuple-inline-dto.json').read_text())
if checked:
    # Direct HTTP object values retain their existing Map projection. Constrain
    # only actual selected DTO sources in models/Patch, not that projection.
    def constrain(v):
        if isinstance(v,dict):
            if 'properties' in v and 'name' in v['properties'] and 'n' in v['properties']:
                v['properties']['name']['minLength']=1
                v['properties']['n']['minimum']=0
                v['minProperties']=1
            for child in v.values(): constrain(child)
        elif isinstance(v,list):
            for child in v: constrain(child)
    constrain(d['components'])
    constrain(d['paths']['/patch'])
    d['components']['schemas']['Counted']['prefixItems'][0]['minProperties']=2
# Independent shallow 240-field probe; it does not stand for width x depth.
d['components']['schemas']['Wide']={'properties':{f'v{i:03d}':{'type':'array','prefixItems':[{'properties':{'name':{'type':'string'}},'required':['name']}],'default':[{'name':'wide'}]} for i in range(240)}}
(p/'input.json').write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
consumer=p/'src/main/kotlin/org/example/contracts/TupleInlineDtoContracts.kt'
(p/'src/main/kotlin/org/example/contracts/OptimizedTupleInlineDtoContracts.kt').write_text(consumer.read_text().replace('org.example.inlinedtotuples','org.example.inlinedtotuplesoptimized').replace('InlineDtoTuple','OptimizedInlineDtoTuple').replace('inlineDtoTuple','optimizedInlineDtoTuple').replace('tupleInlineDto','optimizedTupleInlineDto'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
    tupleInlineDtoContracts('''+str(checked).lower()+''')
    optimizedTupleInlineDtoContracts('''+str(checked).lower()+''')
}
''')
build=p/'build.gradle.kts'
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runTupleInlineDtoContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
    for mode in baseline optimized; do
      output="$project/build/generated"
      package=org.example.inlinedtotuples
      options=()
      if [[ "$mode" == optimized ]]; then
        output="$project/build/generated-limits"
        package=org.example.inlinedtotuplesoptimized
        options=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
      "$workspace/openapi-kt" generate --input tests/fixtures/http/tuple-inline-dto-directional.json --output "$output/directional" --package-name "$package.directional" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input tests/fixtures/http/tuple-inline-dto-directional.json --output "$output/directional" --package-name "$package.directional" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
    done
    python3 - "$project/build/generated-limits/.openapi-kt-names.json" <<'PYLOCK'
import json,sys
assert json.load(open(sys.argv[1]))['entries'], 'optimized tuple profile requires actual locked hoist allocations'
PYLOCK
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleInlineDtoContracts
    echo "FRESH INLINE DTO TUPLE PROFILE $style $checked PASS: $project"
  done
done
