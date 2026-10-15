#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/enum-owner-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/EnumNominalOwnersContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    cp tests/fixtures/http/enum-nominal-owners.json "$project/input.json"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.enumowners --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.enumowners --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
vocabulary={'enum':['v'+str(i) for i in range(240)],'nullable':True}
s={'Owner':{'properties':{'state':vocabulary}},'Named':vocabulary,'Alias':{'$ref':r+'Owner/properties/state'},'False':{'$ref':r+'Alias','nullable':False},'Wide':{'properties':{'p'+str(i):{'$ref':r+'Alias'} for i in range(240)},'required':['p'+str(i) for i in range(240)]},'Hops':{'properties':{}}}
for i in range(47):
 for suffix,target in [('n','Named'),('i','Owner/properties/state')]:s['Hops']['properties'][suffix+str(i)]={'$ref':r+(target if i==0 else 'Hops/properties/'+suffix+str(i-1))}
s['DeepNamed']={'$ref':r+'Hops/properties/n46'};s['DeepInternal']={'$ref':r+'Hops/properties/i46'}
s['TypedNamed']={'$ref':r+'Hops/properties/n44','nullable':False};s['TypedInternal']={'$ref':r+'Hops/properties/i44','nullable':False}
s['Consumer']={'properties':{'named':{'$ref':r+'TypedNamed','nullable':False},'internal':{'$ref':r+'TypedInternal','nullable':False}},'required':['named','internal']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.enumownerlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else 'check(org.example.enumownerlimits.models.ConsumerDto(org.example.enumownerlimits.models.NamedDto.V0, org.example.enumownerlimits.models.OwnerDto.State.V239).copy(internal = org.example.enumownerlimits.models.OwnerDto.State.V0).internal == org.example.enumownerlimits.models.OwnerDto.State.V0)'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(r'''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.serializer
fun main()=runBlocking {
    '''+copy+r'''
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val json=Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(org.example.enumownerlimits.models.OwnerDto.State.entries.size == 240)
        check(org.example.enumownerlimits.models.NamedDto.entries.size == 240)
        check(json.decodeFromString<org.example.enumownerlimits.models.DeepInternalDto>("null") == null)
        check(json.decodeFromString<org.example.enumownerlimits.models.DeepNamedDto>("null") == null)
        val internal: org.example.enumownerlimits.models.TypedInternalDto=org.example.enumownerlimits.models.OwnerDto.State.Unrecognized("future")
        val named: org.example.enumownerlimits.models.TypedNamedDto=org.example.enumownerlimits.models.NamedDto.V239
        val made=org.example.enumownerlimits.models.ConsumerDto(named,internal)
        check(json.decodeFromString<org.example.enumownerlimits.models.ConsumerDto>(json.encodeToString(made)) == made)
        check(runCatching { json.decodeFromString<org.example.enumownerlimits.models.ConsumerDto>("{\"named\":null,\"internal\":null}") }.isFailure)
        val wide=json.decodeFromJsonElement<org.example.enumownerlimits.models.WideDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(wide.p0 == null && wide.p239 == null)
        check(serializer<org.example.enumownerlimits.models.AliasDto>().descriptor.isNullable)
        check(!serializer<org.example.enumownerlimits.models.FalseDto>().descriptor.isNullable)
    }
    enumNominalOwnersContracts()
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runEnumNominalOwnersContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runEnumNominalOwnersContracts
    echo "FRESH ENUM NOMINAL OWNERS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh enum nominal owner native projects: $workspace"
