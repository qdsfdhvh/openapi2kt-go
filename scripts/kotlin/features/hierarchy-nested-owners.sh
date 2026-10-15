#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/hierarchy-nested-owner-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/HierarchyNestedOwnersContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    cp tests/fixtures/http/hierarchy-nested-owners.json "$project/input.json"
    if [[ "$checked" == on ]]; then
      python3 - "$project/input.json" <<'PYCHECKS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(p.read_text());s=d['components']['schemas']
for name in ('First','Second'):s[name]['properties']['node']['properties']['id']['minLength']=1
s['First']['properties']['rows']['items']['properties']['id']['minLength']=1
s['First']['properties']['entries']['additionalProperties']['properties']['id']['minLength']=1
p.write_text(json.dumps(d))
PYCHECKS
    fi
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.hierarchynested --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.hierarchynested --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
schemas={
 'Root':{'oneOf':[{'type':'null'},{'$ref':r+'First','nullable':True},{'$ref':r+'Second'}],'discriminator':{'propertyName':'kind','mapping':{'first':'First','second':'Second'}}},
 'First':{'nullable':True,'properties':{'kind':{'type':'string','const':'first'},'id':{'type':'string'}},'required':['id']},
 'Second':{'properties':{'kind':{'type':'string','const':'second'},'id':{'type':'string'}},'required':['id']},
 'EmptyRoot':{'oneOf':[{'type':'null'},{'$ref':r+'Empty'},{'$ref':r+'OtherEmpty'}],'discriminator':{'propertyName':'tag'}},
 'Empty':{'type':'object'},
 'OtherEmpty':{'properties':{'id':{'type':'string'}}},
 'EmptyAlias':{'$ref':r+'Empty','nullable':True},
 'EmptyFalse':{'$ref':r+'Empty','nullable':False},
 'EmptyHolder':{'properties':{'value':{'$ref':r+'EmptyAlias'},'narrow':{'$ref':r+'EmptyFalse'}},'required':['value','narrow']},
 'HopOwner':{'properties':{'parent':{'$ref':r+'Root'},'leaf':{'$ref':r+'First'}}},
 'Nullable':{'$ref':r+'HopOwner/properties/parent'},
 'LeafBridge':{'$ref':r+'HopOwner/properties/leaf'},
 'Wide':{'properties':{'p'+str(i):{'$ref':r+'Nullable'} for i in range(240)},'required':['p'+str(i) for i in range(240)]},
 'RootLimit':{'properties':{'id':{'type':'string'},'node':{'properties':{'id':{'type':'string'}},'required':['id']}},'required':['id']},
 'NodeAlias':{'$ref':r+'RootLimit/properties/node','nullable':True},
 'NestedWide':{'properties':{'p'+str(i):{'$ref':r+'NodeAlias'} for i in range(240)},'required':['p'+str(i) for i in range(240)]},
 'OtherLimit':{'properties':{'id':{'type':'string'}},'required':['id']},
 'HierarchyLimit':{'oneOf':[{'type':'null'},{'$ref':r+'RootLimit'},{'$ref':r+'OtherLimit'}],'discriminator':{'propertyName':'tag','mapping':{'root':'RootLimit','other':'OtherLimit'}}},
}
schemas['InternalOwner']={'properties':{}}
for i in range(47):
 schemas['InternalOwner']['properties']['h'+str(i)]={'$ref':r+('HierarchyLimit' if i==0 else 'InternalOwner/properties/h'+str(i-1))}
schemas['Deep']={'$ref':r+'InternalOwner/properties/h46'}
schemas['Typed']={'$ref':r+'InternalOwner/properties/h44','nullable':False}
schemas['Consumer']={'properties':{'narrowed':{'$ref':r+'Typed','nullable':False}},'required':['narrowed']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.hierarchynestedlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else '''val original = org.example.hierarchynested.models.AFirstDto("copy")
    check(original.copy(id="updated") == org.example.hierarchynested.models.FirstDto("updated"))'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.serializer
fun main()=runBlocking {
    '''+copy+'''
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val json = Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(serializer<org.example.hierarchynestedlimits.models.DeepDto>().descriptor.isNullable)
        check(json.decodeFromString<org.example.hierarchynestedlimits.models.DeepDto>("null") == null)
        val made = org.example.hierarchynestedlimits.models.RootLimitDto("deep")
        val value = org.example.hierarchynestedlimits.models.ConsumerDto(made)
        check(json.decodeFromJsonElement<org.example.hierarchynestedlimits.models.ConsumerDto>(json.encodeToJsonElement(value)) == value)
        check(runCatching { json.decodeFromString<org.example.hierarchynestedlimits.models.ConsumerDto>("{\\"narrowed\\":null}") }.isFailure)
        val wide = json.decodeFromJsonElement<org.example.hierarchynestedlimits.models.WideDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(wide.p0 == null && wide.p239 == null)
        val nestedWide = json.decodeFromJsonElement<org.example.hierarchynestedlimits.models.NestedWideDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(nestedWide.p0 == null && nestedWide.p239 == null)
        check(serializer<org.example.hierarchynestedlimits.models.EmptyAliasDto>().descriptor.isNullable)
        check(!serializer<org.example.hierarchynestedlimits.models.EmptyFalseDto>().descriptor.isNullable)
        check(json.decodeFromString<org.example.hierarchynestedlimits.models.EmptyAliasDto>("null") == null)
        check(runCatching { json.decodeFromString(org.example.hierarchynestedlimits.models.EmptyDto.serializer(), "null") }.isFailure)
        check(runCatching { json.decodeFromString<org.example.hierarchynestedlimits.models.EmptyFalseDto>("null") }.isFailure)
        check(runCatching { json.decodeFromString<org.example.hierarchynestedlimits.models.EmptyHolderDto>("{\\"value\\":null,\\"narrow\\":null}") }.isFailure)
        val empty = org.example.hierarchynestedlimits.models.EmptyDto
        val holder = org.example.hierarchynestedlimits.models.EmptyHolderDto(null, empty)
        check(json.decodeFromJsonElement<org.example.hierarchynestedlimits.models.EmptyHolderDto>(json.encodeToJsonElement(holder)) == holder)
        check(json.encodeToJsonElement(empty) == json.parseToJsonElement("{\\"tag\\":\\"empty\\"}"))
        check(json.decodeFromString<org.example.hierarchynestedlimits.models.EmptyRootDto>("{\\"tag\\":\\"empty\\"}") == empty)
    }
    hierarchyNestedOwnersContracts(checked = '''+str(sys.argv[3]=='on').lower()+''')
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runHierarchyNestedOwnersContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runHierarchyNestedOwnersContracts
    echo "FRESH HIERARCHY NESTED OWNERS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh hierarchy nested owner native projects: $workspace"
