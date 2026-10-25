#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/pinned-structured-const-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/PinnedStructuredConstContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  validation=(); [[ "$checked" == off ]] || validation+=(--validation)
  for mode in base optimized; do
   output="$project/build/generated";package=org.example.pinned;options=()
   if [[ "$mode" == optimized ]]; then
    output="$project/build/generated-limits";package+=optimized
    options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
   fi
   args=(--input tests/fixtures/http/pinned-structured-const.json --output "$output" --package-name "$package" --model-name-suffix Dto --model-style "$style" --api-client-name PinnedClient)
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
  done
  python3 - "$project" "$style" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1])
assert json.loads((p/'build/generated-limits/.openapi-kt-names.json').read_text())['entries']
c=(p/'src/main/kotlin/org/example/contracts/PinnedStructuredConstContracts.kt').read_text()
d=p/'src/main/kotlin/org/example/optimizedcontracts';d.mkdir(parents=True)
(d/'PinnedStructuredConstContracts.kt').write_text(c.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.pinned.','org.example.pinnedoptimized.'))
ref=lambda name:{'$ref':'#/components/schemas/'+name}
props={'tag':{'const':'one'}}|{f'p{i:03d}':{'const':{'index':i}} for i in range(239)}
doc={'openapi':'3.1.0','components':{'schemas':{'Wide':{'oneOf':[{'properties':props},{'properties':{'tag':{'const':'two'},'p000':{'const':{}}}}]}}}}
(p/'wide.json').write_text(json.dumps(doc)+'\n')
data_copy='' if sys.argv[2]=='plain_class' else 'check(org.example.pinned.models.WithArgsDto.One(label="original").copy(label="copy").label=="copy")'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import org.example.pinnedwide.models.WideDto
fun main()=runBlocking {
 pinnedStructuredConstContracts()
 org.example.optimizedcontracts.pinnedStructuredConstContracts()
'''+data_copy+'''
 val value=WideDto.One()
 check(value.p000["index"].toString()=="0" && value.p238["index"].toString()=="238")
 check(WideDto.One.serializer().descriptor.elementsCount==240)
 for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
  val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
  val alternative=codec.decodeFromString(WideDto.serializer(),"{\\"tag\\":\\"one\\",\\"p238\\":{\\"alternative\\":null}}") as WideDto.One
  check(alternative.p238.containsKey("alternative"))
  val encoded=codec.encodeToString(WideDto.serializer(),alternative)
  check((codec.decodeFromString(WideDto.serializer(),encoded) as WideDto.One).p238==alternative.p238)
 }
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runPinnedStructuredConstContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.pinnedwide --model-name-suffix Dto --model-style "$style" "${validation[@]}"
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runPinnedStructuredConstContracts
  echo "FRESH PINNED STRUCTURED CONST PROFILE $style-$checked PASS: $project"
 done
done
echo "Fresh pinned structured const projects: $workspace"
