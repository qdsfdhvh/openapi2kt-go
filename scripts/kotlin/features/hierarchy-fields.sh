#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/hierarchy-fields-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/HierarchyFieldsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    "$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-fields.json --output "$project/build/generated" --package-name org.example.hierarchyfields --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-fields.json --output "$project/build/generated" --package-name org.example.hierarchyfields --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
def ref(name): return {'$ref':r+name}
schemas={
 'WideRoot':{'oneOf':[ref('WideLeaf')],'discriminator':{'propertyName':'kind','mapping':{'wide':'WideLeaf'}}},
 'WideLeaf':{'type':'object','properties':{'p'+str(i):{'type':'boolean'} for i in range(239)},'required':['p'+str(i) for i in range(239)]},
 'FixedRoot':{'oneOf':[ref('FixedLeaf')],'discriminator':{'propertyName':'kind','mapping':{'fixed':'FixedLeaf'}}},
 'FixedLeaf':{'properties':{'kind':{'type':'string','const':'fixed'}}},
 'EnumRoot':{'oneOf':[ref('EnumLeaf')],'discriminator':{'propertyName':'kind','mapping':{'enum':'EnumLeaf'}}},
 'EnumLeaf':{'properties':{'kind':{'enum':['enum'],'nullable':True}}},
}
# 48 hierarchy levels retain original edges. Distinct short wire fields avoid
# reused discriminator ownership, with 47 generated leaf bodies counted.
for i in range(47):
 schemas['D'+str(i)]={'oneOf':[ref('D'+str(i+1))],'discriminator':{'propertyName':'t'+str(i),'mapping':{'v'+str(i):'D'+str(i+1)}}}
schemas['D47']={'properties':{'id':{'type':'string'}},'required':['id']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.hierarchyfieldslimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else '''val original = org.example.hierarchyfields.models.MissingFirstDto("copy")
    check(original.copy(id="updated").kind == original.kind)'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.jsonObject
fun main()=runBlocking {
    '''+copy+'''
    for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {
        val json = Json { encodeDefaults=defaults; explicitNulls=nulls }
        val fixed: org.example.hierarchyfieldslimits.models.FixedRootDto = org.example.hierarchyfieldslimits.models.FixedLeafDto
        check(json.decodeFromJsonElement<org.example.hierarchyfieldslimits.models.FixedRootDto>(json.encodeToJsonElement(fixed)) === fixed)
        val enum = org.example.hierarchyfieldslimits.models.EnumLeafDto(org.example.hierarchyfieldslimits.models.EnumRootDtoKind.ENUM)
        check(json.decodeFromJsonElement<org.example.hierarchyfieldslimits.models.EnumRootDto>(json.encodeToJsonElement<org.example.hierarchyfieldslimits.models.EnumRootDto>(enum)) == enum)
        check(runCatching { json.decodeFromString<org.example.hierarchyfieldslimits.models.EnumLeafDto>("{}") }.isFailure)
        val deep: org.example.hierarchyfieldslimits.models.D0Dto = org.example.hierarchyfieldslimits.models.D47Dto("deep")
        val wire = json.encodeToJsonElement(deep)
        check(wire.jsonObject.size == 48)
        check(json.decodeFromJsonElement<org.example.hierarchyfieldslimits.models.D0Dto>(wire) == deep)
    }
    hierarchyFieldsContracts()
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runHierarchyFieldsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runHierarchyFieldsContracts
    echo "FRESH HIERARCHY FIELDS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh hierarchy fields native projects: $workspace"
