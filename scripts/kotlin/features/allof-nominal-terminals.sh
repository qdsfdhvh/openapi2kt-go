#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/allof-terminal-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/AllOfNominalTerminalsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    cp tests/fixtures/http/allof-nominal-terminals.json "$project/input.json"
    if [[ "$checked" == on ]]; then
      python3 - "$project/input.json" <<'PYCHECKS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(p.read_text());s=d['components']['schemas']
s['AllModel']['allOf'][0]['properties']['id']['minLength']=1
s['AllContainer']['properties']['node']['allOf'][0]['properties']['id']['minLength']=1
p.write_text(json.dumps(d))
PYCHECKS
    fi
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.allofterminals --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.allofterminals --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
s={
 'Owner':{'properties':{'node':{'allOf':[{'properties':{'id':{'type':'string'}},'required':['id']},{'properties':{'next':{'$ref':r+'NodeAlias'}}}]}}},
 'AllModel':{'allOf':[{'properties':{'id':{'type':'string'}},'required':['id']},{'properties':{'node':{'$ref':r+'NodeAlias'}}}]},
 'NodeAlias':{'$ref':r+'Owner/properties/node','nullable':True},
 'Wide':{'properties':{'p'+str(i):{'$ref':r+'NodeAlias'} for i in range(240)},'required':['p'+str(i) for i in range(240)]},
 'Hops':{'properties':{}},
}
for i in range(47):
 for suffix,target in [('n','AllModel'),('i','Owner/properties/node')]:s['Hops']['properties'][suffix+str(i)]={'$ref':r+(target if i==0 else 'Hops/properties/'+suffix+str(i-1))}
s['DeepNamed']={'$ref':r+'Hops/properties/n46'}
s['DeepInternal']={'$ref':r+'Hops/properties/i46'}
s['TypedNamed']={'$ref':r+'Hops/properties/n44','nullable':False}
s['TypedInternal']={'$ref':r+'Hops/properties/i44','nullable':False}
s['Consumer']={'properties':{'named':{'$ref':r+'TypedNamed','nullable':False},'internal':{'$ref':r+'TypedInternal','nullable':False}},'required':['named','internal']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.allofterminallimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else 'check(org.example.allofterminals.models.AllNestedDto("original").copy(id="changed") == org.example.allofterminals.models.AllContainerDto.Node("changed"))'
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
        val node = org.example.allofterminallimits.models.DeepInternalDto("deep")
        val named = org.example.allofterminallimits.models.DeepNamedDto("named",node)
        check(node == org.example.allofterminallimits.models.OwnerDto.Node("deep"))
        check(named == org.example.allofterminallimits.models.AllModelDto("named",node))
        val made = org.example.allofterminallimits.models.ConsumerDto(named,node)
        check(json.decodeFromString<org.example.allofterminallimits.models.ConsumerDto>(json.encodeToString(made)) == made)
        check(runCatching { json.decodeFromString<org.example.allofterminallimits.models.ConsumerDto>("{\"named\":null,\"internal\":null}") }.isFailure)
        val wide = json.decodeFromJsonElement<org.example.allofterminallimits.models.WideDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(wide.p0 == null && wide.p239 == null)
        check(serializer<org.example.allofterminallimits.models.NodeAliasDto>().descriptor.isNullable)
        check(!serializer<org.example.allofterminallimits.models.TypedInternalDto>().descriptor.isNullable)
    }
    allOfNominalTerminalsContracts(checked = '''+str(sys.argv[3]=='on').lower()+r''')
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runAllOfNominalTerminalsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runAllOfNominalTerminalsContracts
    echo "FRESH ALLOF NOMINAL TERMINALS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh allOf nominal terminal native projects: $workspace"
