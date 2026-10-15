#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/hierarchy-intrinsic-nullable-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/HierarchyIntrinsicNullableContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    "$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-intrinsic-nullable.json --output "$project/build/generated" --package-name org.example.hierarchyintrinsic --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-intrinsic-nullable.json --output "$project/build/generated" --package-name org.example.hierarchyintrinsic --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
schemas={
 'Root':{'nullable':True,'oneOf':[{'$ref':r+'First'},{'$ref':r+'Second'}],'discriminator':{'propertyName':'kind','mapping':{'first':'First','second':'Second'}}},
 'First':{'nullable':True,'properties':{'kind':{'type':'string','const':'first'},'id':{'type':'string'}},'required':['id']},
 'Second':{'properties':{'kind':{'type':'string','const':'second'},'id':{'type':'string'}},'required':['id']},
 'EmptyRoot':{'oneOf':[{'$ref':r+'Empty'},{'$ref':r+'OtherEmpty'}],'discriminator':{'propertyName':'tag'}},
 'Empty':{'type':'object','nullable':True},
 'OtherEmpty':{'properties':{'id':{'type':'string'}}},
 'EmptyAlias':{'$ref':r+'Empty'},
 'EmptyFalse':{'$ref':r+'Empty','nullable':False},
 'EmptyHolder':{'properties':{'value':{'$ref':r+'Empty'},'narrow':{'$ref':r+'EmptyFalse'}},'required':['value','narrow']},
 'Nullable':{'$ref':r+'Root'},
 'Wide':{'properties':{'p'+str(i):{'$ref':r+'Nullable'} for i in range(240)},'required':['p'+str(i) for i in range(240)]},
 'RootLimit':{'properties':{'id':{'type':'string'}},'required':['id']},
 'OtherLimit':{'properties':{'id':{'type':'string'}},'required':['id']},
 'HierarchyLimit':{'nullable':True,'oneOf':[{'$ref':r+'RootLimit'},{'$ref':r+'OtherLimit'}],'discriminator':{'propertyName':'tag','mapping':{'root':'RootLimit','other':'OtherLimit'}}},
}
for i in range(48):
 schemas['A'+str(i)]={'$ref':r+('HierarchyLimit' if i==0 else 'A'+str(i-1))}
# Two use-site edges plus 46 original alias edges reach the source48 budget.
schemas['Consumer']={'properties':{'narrowed':{'$ref':r+'A45','nullable':False}},'required':['narrowed']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.hierarchyintrinsiclimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else '''val original = org.example.hierarchyintrinsic.models.AFirstDto("copy")
    check(original.copy(id="updated") == org.example.hierarchyintrinsic.models.FirstDto("updated"))'''
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
        check(serializer<org.example.hierarchyintrinsiclimits.models.A47Dto>().descriptor.isNullable)
        check(json.decodeFromString<org.example.hierarchyintrinsiclimits.models.A47Dto>("null") == null)
        val made = org.example.hierarchyintrinsiclimits.models.RootLimitDto("deep")
        val value = org.example.hierarchyintrinsiclimits.models.ConsumerDto(made)
        check(json.decodeFromJsonElement<org.example.hierarchyintrinsiclimits.models.ConsumerDto>(json.encodeToJsonElement(value)) == value)
        check(runCatching { json.decodeFromString<org.example.hierarchyintrinsiclimits.models.ConsumerDto>("{\\"narrowed\\":null}") }.isFailure)
        val wide = json.decodeFromJsonElement<org.example.hierarchyintrinsiclimits.models.WideDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(wide.p0 == null && wide.p239 == null)
        check(serializer<org.example.hierarchyintrinsiclimits.models.EmptyAliasDto>().descriptor.isNullable)
        check(!serializer<org.example.hierarchyintrinsiclimits.models.EmptyFalseDto>().descriptor.isNullable)
        check(json.decodeFromString<org.example.hierarchyintrinsiclimits.models.EmptyAliasDto>("null") == null)
        check(runCatching { json.decodeFromString(org.example.hierarchyintrinsiclimits.models.EmptyDto.serializer(), "null") }.isFailure)
        check(runCatching { json.decodeFromString<org.example.hierarchyintrinsiclimits.models.EmptyFalseDto>("null") }.isFailure)
        check(runCatching { json.decodeFromString<org.example.hierarchyintrinsiclimits.models.EmptyHolderDto>("{\\"value\\":null,\\"narrow\\":null}") }.isFailure)
        val empty = org.example.hierarchyintrinsiclimits.models.EmptyDto
        val holder = org.example.hierarchyintrinsiclimits.models.EmptyHolderDto(null, empty)
        check(json.decodeFromJsonElement<org.example.hierarchyintrinsiclimits.models.EmptyHolderDto>(json.encodeToJsonElement(holder)) == holder)
        check(json.encodeToJsonElement(empty) == json.parseToJsonElement("{\\"tag\\":\\"empty\\"}"))
        check(json.decodeFromString<org.example.hierarchyintrinsiclimits.models.EmptyRootDto>("{\\"tag\\":\\"empty\\"}") == empty)
    }
    hierarchyIntrinsicNullableContracts()
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runHierarchyIntrinsicNullableContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runHierarchyIntrinsicNullableContracts
    echo "FRESH HIERARCHY INTRINSIC NULLABLE PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh hierarchy intrinsic nullable native projects: $workspace"
