#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-inline-allof-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleInlineAllOfContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/tuple-inline-allof.json').read_text())
if checked:
    def constrain(v):
        if isinstance(v,dict):
            props=v.get('properties',{})
            if 'name' in props: props['name']['minLength']=1
            if 'n' in props: props['n']['minimum']=0
            for child in v.values(): constrain(child)
        elif isinstance(v,list):
            for child in v: constrain(child)
    constrain(d['components'])
    constrain(d['paths']['/patch'])
    d['components']['schemas']['Counted']['prefixItems'][0]['minProperties']=2
# Independent 240 fields exercise real allOf declarations and typed own defaults.
d['components']['schemas']['Wide']={'properties':{f'v{i:03d}':{'type':'array','prefixItems':[{'allOf':[{'properties':{'name':{'type':'string'}},'required':['name']},{'properties':{'n':{'type':'integer'}}}]}],'default':[{'name':'wide'}]} for i in range(240)}}
(p/'input.json').write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
consumer=p/'src/main/kotlin/org/example/contracts/TupleInlineAllOfContracts.kt'
(p/'src/main/kotlin/org/example/contracts/OptimizedTupleInlineAllOfContracts.kt').write_text(consumer.read_text().replace('org.example.inlineallof','org.example.inlineallofoptimized').replace('InlineAllOfTuple','OptimizedInlineAllOfTuple').replace('inlineAllOfTuple','optimizedInlineAllOfTuple').replace('tupleInlineAllOf','optimizedTupleInlineAllOf'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
    tupleInlineAllOfContracts('''+str(checked).lower()+''')
    optimizedTupleInlineAllOfContracts('''+str(checked).lower()+''')
}
''')
build=p/'build.gradle.kts'
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runTupleInlineAllOfContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
    for mode in baseline optimized; do
      output="$project/build/generated"
      package=org.example.inlineallof
      options=()
      if [[ "$mode" == optimized ]]; then
        output="$project/build/generated-limits"
        package=org.example.inlineallofoptimized
        options=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
    done
    python3 - "$project/build/generated-limits/.openapi-kt-names.json" <<'PYLOCK'
import json,sys
assert json.load(open(sys.argv[1]))['entries'], 'optimized tuple profile requires actual locked hoist allocations'
PYLOCK
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleInlineAllOfContracts
    echo "FRESH INLINE ALLOF TUPLE PROFILE $style $checked PASS: $project"
  done
done
