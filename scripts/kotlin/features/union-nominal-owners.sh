#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/union-owner-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnionNominalOwnersContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    cp tests/fixtures/http/union-nominal-owners.json "$project/input.json"
    if [[ "$checked" == on ]]; then
      python3 - "$project/input.json" <<'PYCHECKS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(p.read_text());s=d['components']['schemas']
s['UnionBox']['properties']['choice']['oneOf'][0]['properties']['id']['minLength']=1
p.write_text(json.dumps(d))
PYCHECKS
    fi
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.unionowners --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.unionowners --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
union={'oneOf':[{'type':'string'},{'type':'boolean'},{'type':'null'}]}
s={
 'Owner':{'properties':{'choice':union}},
 'Carrier':union,
 'Inner':{'$ref':r+'Owner/properties/choice'},
 'InnerFalse':{'$ref':r+'Inner','nullable':False},
 'NullableCarrier':{'$ref':r+'Carrier','nullable':True},
 'CarrierFalse':{'$ref':r+'NullableCarrier','nullable':False},
 'Wide':{'properties':{'p'+str(i):{'$ref':r+'Inner'} for i in range(240)},'required':['p'+str(i) for i in range(240)]},
 'Hops':{'properties':{}},
 'Branches':{'properties':{'choice':{'oneOf':[{'properties':{'kind':{'type':'string','const':'v'+str(i)},'id':{'type':'string'}},'required':['id']} for i in range(240)]}}},
 'BranchAlias':{'$ref':r+'Branches/properties/choice'},
}
for i in range(47):
 for suffix,target in [('n','Carrier'),('i','Owner/properties/choice')]:s['Hops']['properties'][suffix+str(i)]={'$ref':r+(target if i==0 else 'Hops/properties/'+suffix+str(i-1))}
s['DeepNamed']={'$ref':r+'Hops/properties/n46'}
s['DeepInternal']={'$ref':r+'Hops/properties/i46'}
s['TypedNamed']={'$ref':r+'Hops/properties/n44','nullable':False}
s['TypedInternal']={'$ref':r+'Hops/properties/i44','nullable':False}
s['Consumer']={'properties':{'named':{'$ref':r+'TypedNamed','nullable':False},'internal':{'$ref':r+'TypedInternal','nullable':False}},'required':['named','internal']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.unionownerlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else 'check(org.example.unionowners.models.UnionBoxDto.Choice.A("original").copy(id="changed") == org.example.unionowners.models.UnionBoxDto.Choice.A("changed"))'
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
        val json = Json { encodeDefaults=defaults; explicitNulls=nulls }
        val internal: org.example.unionownerlimits.models.TypedInternalDto = org.example.unionownerlimits.models.OwnerDto.BooleanChoice(true)
        val named: org.example.unionownerlimits.models.TypedNamedDto = org.example.unionownerlimits.models.BooleanCarrierDto(false)
        check(json.decodeFromString<org.example.unionownerlimits.models.DeepInternalDto>("null") == null)
        check(json.decodeFromString<org.example.unionownerlimits.models.DeepNamedDto>("null") == org.example.unionownerlimits.models.NullCarrierDto)
        val made = org.example.unionownerlimits.models.ConsumerDto(named,internal)
        check(json.decodeFromString<org.example.unionownerlimits.models.ConsumerDto>(json.encodeToString(made)) == made)
        check(runCatching { json.decodeFromString<org.example.unionownerlimits.models.ConsumerDto>("{\"named\":null,\"internal\":null}") }.isFailure)
        val wide = json.decodeFromJsonElement<org.example.unionownerlimits.models.WideDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(wide.p0 == null && wide.p239 == null)
        check(serializer<org.example.unionownerlimits.models.InnerDto>().descriptor.isNullable)
        check(!serializer<org.example.unionownerlimits.models.TypedInternalDto>().descriptor.isNullable)
        val branch: org.example.unionownerlimits.models.BranchAliasDto = org.example.unionownerlimits.models.BranchesDto.Choice.V239("last")
        check(json.decodeFromString<org.example.unionownerlimits.models.BranchAliasDto>(json.encodeToString(branch)) == branch)
    }
    unionNominalOwnersContracts(checked = '''+str(sys.argv[3]=='on').lower()+r''')
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runUnionNominalOwnersContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runUnionNominalOwnersContracts
    echo "FRESH UNION NOMINAL OWNERS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh union nominal owner native projects: $workspace"
