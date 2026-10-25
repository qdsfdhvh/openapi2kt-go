#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/discarded-structured-const-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DiscardedStructuredConstContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  validation=(); [[ "$checked" == off ]] || validation+=(--validation)
  for mode in base optimized; do
   output="$project/build/generated";package=org.example.discarded;options=()
   if [[ "$mode" == optimized ]]; then
    output="$project/build/generated-limits";package+=optimized
    options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
   fi
   args=(--input tests/fixtures/http/discarded-structured-const.json --output "$output" --package-name "$package" --model-name-suffix Dto --model-style "$style" --api-client-name DiscardedClient)
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
  done
  python3 - "$project" "$style" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);assert json.loads((p/'build/generated-limits/.openapi-kt-names.json').read_text())['entries']
c=(p/'src/main/kotlin/org/example/contracts/DiscardedStructuredConstContracts.kt').read_text()
d=p/'src/main/kotlin/org/example/optimizedcontracts';d.mkdir(parents=True)
(d/'DiscardedStructuredConstContracts.kt').write_text(c.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.discarded.','org.example.discardedoptimized.'))
fields={f'p{i:03d}':{'type':'string'} for i in range(240)}
large={'literal':'a'*65536}
pinned={'tag':{'const':'one'}}|{f'p{i:03d}':{'type':'string'} for i in range(239)}
s={'Wide':{'properties':fields,'required':list(fields),'additionalProperties':{'const':large}},'WidePinned':{'oneOf':[{'properties':pinned,'required':list(pinned),'additionalProperties':{'const':[None]}},{'properties':{'tag':{'const':'two'}},'additionalProperties':{'const':{}}}]}}
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s}})+'\n')
args=','.join(['"value"']*240);pinnedargs=','.join(['"value"']*239)
copy='' if sys.argv[2]=='plain_class' else 'check(org.example.discarded.models.ChoiceDto.One(payload="original").copy(payload="copy").payload=="copy")'
main='''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import org.example.discardedwide.models.WideDto
import org.example.discardedwide.models.WidePinnedDto
fun main()=runBlocking {
 discardedStructuredConstContracts()
 org.example.optimizedcontracts.discardedStructuredConstContracts()
'''+copy+'\n val wide=WideDto('+args+')\n val pinned=WidePinnedDto.One('+pinnedargs+')\n'+'''
 check(wide.p000=="value" && wide.p239=="value")
 check(WideDto.serializer().descriptor.elementsCount==240)
 check(WidePinnedDto.One.serializer().descriptor.elementsCount==240)
 for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
  val codec=Json {encodeDefaults=defaults;explicitNulls=nulls;ignoreUnknownKeys=true}
  val encoded=codec.encodeToString(WideDto.serializer(),wide)
  check(codec.decodeFromString(WideDto.serializer(),encoded).p239=="value")
  check((codec.decodeFromString(WidePinnedDto.serializer(),codec.encodeToString(WidePinnedDto.serializer(),pinned)) as WidePinnedDto.One).p238=="value")
 }
}
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(main)
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runDiscardedStructuredConstContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.discardedwide --model-name-suffix Dto --model-style "$style" "${validation[@]}"
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runDiscardedStructuredConstContracts
  echo "FRESH DISCARDED STRUCTURED CONST PROFILE $style-$checked PASS: $project"
 done
done
echo "Fresh discarded structured const projects: $workspace"
