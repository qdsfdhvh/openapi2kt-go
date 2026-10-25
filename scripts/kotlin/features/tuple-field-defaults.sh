#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-field-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for validation in off on; do
    project="$workspace/$style-$validation"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleFieldDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$validation" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(Path('tests/fixtures/http/tuple-field-defaults.json').read_text())
if sys.argv[2]=='on':
 field=d['components']['schemas']['Holder']['properties']['longs']
 field.update(minItems=1,maxItems=3,uniqueItems=True,items=dict(field['prefixItems'][0]))
d['components']['schemas']['Wide']={'properties':{'p'+str(i):{'type':'array','prefixItems':[{'type':'integer'}],'default':[1]} for i in range(240)}}
(p/'input.json').write_text(json.dumps(d)+'\n')
PYINPUT
    flags=()
    [[ "$validation" == off ]] || flags+=(--validation)
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.tuplefielddefaults --api-client-name TupleFieldDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.tuplefielddefaults --api-client-name TupleFieldDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.tuplefielddefaultsoptimized --api-client-name TupleFieldDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.tuplefielddefaultsoptimized --api-client-name TupleFieldDefaults --model-name-suffix Dto --model-style "$style" "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names --check
    python3 - "$project" "$style" "$validation" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1])
consumer=(p/'src/main/kotlin/org/example/contracts/TupleFieldDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'TupleFieldDefaultsContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.tuplefielddefaults','org.example.tuplefielddefaultsoptimized'))
checks='' if sys.argv[3]=='off' else '''
 check(runCatching { org.example.tuplefielddefaults.models.HolderDto(required=listOf("given"),longs=emptyList()) }.isFailure)
 check(runCatching { org.example.tuplefielddefaults.models.HolderDto(required=listOf("given"),longs=listOf(1,1)) }.isFailure)
'''
copy='' if sys.argv[2]=='plain_class' else '''
 val value=org.example.tuplefielddefaults.models.HolderDto(required=listOf("provided"))
 check(value.copy(kotlin="copy").longs==value.longs)
 check(value.copy(kotlin="copy").dates==value.dates && value.copy().times==value.times)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
 check(org.example.tuplefielddefaults.models.WideDto().p239 == listOf(1))
'''+copy+checks+''' tupleFieldDefaultsContracts(); org.example.optimizedcontracts.tupleFieldDefaultsContracts() }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runTupleFieldDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleFieldDefaultsContracts
    echo "FRESH TUPLE FIELD DEFAULTS PROFILE $style-$validation PASS: $project"
  done
done
echo "Fresh tuple field default native projects: $workspace"
