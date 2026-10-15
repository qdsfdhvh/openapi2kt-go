#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/composed-owner-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ComposedNestedOwnersContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    cp tests/fixtures/http/composed-nested-owners.json "$project/input.json"
    if [[ "$checked" == on ]]; then
      python3 - "$project/input.json" <<'PYCHECKS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(p.read_text());s=d['components']['schemas']
s['Composed']['allOf'][0]['properties']['node']['properties']['id']['minLength']=1
s['Choice']['oneOf'][0]['properties']['node']['properties']['id']['minLength']=1
p.write_text(json.dumps(d))
PYCHECKS
    fi
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.composedowners --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.composedowners --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
s={
 'Owner':{'allOf':[{'properties':{'node':{'properties':{'id':{'type':'string'}},'required':['id']}}},{'properties':{'extra':{'type':'string'}}}]},
 'NodeAlias':{'$ref':r+'Owner/allOf/0/properties/node','nullable':True},
 'Wide':{'properties':{'p'+str(i):{'$ref':r+'NodeAlias'} for i in range(240)},'required':['p'+str(i) for i in range(240)]},
 'Hops':{'properties':{}},
}
for i in range(47):s['Hops']['properties']['h'+str(i)]={'$ref':r+('Owner/allOf/0/properties/node' if i==0 else 'Hops/properties/h'+str(i-1))}
s['Deep']={'$ref':r+'Hops/properties/h46'}
s['Typed']={'$ref':r+'Hops/properties/h44','nullable':False}
s['Consumer']={'properties':{'value':{'$ref':r+'Typed','nullable':False}},'required':['value']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.composedownerlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else 'check(org.example.composedowners.models.ComposedNodeDto("original").copy(id="changed") == org.example.composedowners.models.ComposedDto.Node("changed"))'
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
        val node = org.example.composedownerlimits.models.DeepDto("deep")
        check(node == org.example.composedownerlimits.models.OwnerDto.Node("deep"))
        val made = org.example.composedownerlimits.models.ConsumerDto(node)
        check(json.decodeFromString<org.example.composedownerlimits.models.ConsumerDto>(json.encodeToString(made)) == made)
        check(runCatching { json.decodeFromString<org.example.composedownerlimits.models.ConsumerDto>("{\"value\":null}") }.isFailure)
        val wide = json.decodeFromJsonElement<org.example.composedownerlimits.models.WideDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(wide.p0 == null && wide.p239 == null)
        check(serializer<org.example.composedownerlimits.models.NodeAliasDto>().descriptor.isNullable)
        check(!serializer<org.example.composedownerlimits.models.TypedDto>().descriptor.isNullable)
    }
    composedNestedOwnersContracts(checked = '''+str(sys.argv[3]=='on').lower()+r''')
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runComposedNestedOwnersContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runComposedNestedOwnersContracts
    echo "FRESH COMPOSED NESTED OWNERS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh composed nested owner native projects: $workspace"
