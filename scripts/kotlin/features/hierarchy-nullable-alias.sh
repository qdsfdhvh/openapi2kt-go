#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/hierarchy-nullable-alias-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/HierarchyNullableAliasContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    "$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-nullable-aliases.json --output "$project/build/generated" --package-name org.example.hierarchynullable --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-nullable-aliases.json --output "$project/build/generated" --package-name org.example.hierarchynullable --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
schemas={
 'Root':{'oneOf':[{'$ref':r+'First'},{'$ref':r+'Second'}],'discriminator':{'propertyName':'kind','mapping':{'first':'First','second':'Second'}}},
 'First':{'properties':{'kind':{'type':'string','const':'first'},'id':{'type':'string'}},'required':['id']},
 'Second':{'properties':{'kind':{'type':'string','const':'second'},'id':{'type':'string'}},'required':['id']},
 'Nullable':{'$ref':r+'Root','nullable':True},
 'Wide':{'properties':{'p'+str(i):{'$ref':r+'Nullable'} for i in range(240)},'required':['p'+str(i) for i in range(240)]},
 'RootLimit':{'properties':{'id':{'type':'string'}},'required':['id']},
 'OtherLimit':{'properties':{'id':{'type':'string'}},'required':['id']},
 'HierarchyLimit':{'oneOf':[{'$ref':r+'RootLimit'},{'$ref':r+'OtherLimit'}],'discriminator':{'propertyName':'tag','mapping':{'root':'RootLimit','other':'OtherLimit'}}},
}
for i in range(48):
 schemas['A'+str(i)]={'$ref':r+('HierarchyLimit' if i==0 else 'A'+str(i-1))}
 if i==0:schemas['A0']['nullable']=True
# Two use-site edges plus 46 original alias edges reach the source48 budget.
schemas['Consumer']={'properties':{'narrowed':{'$ref':r+'A45','nullable':False}},'required':['narrowed']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.hierarchynullablelimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else '''val original = org.example.hierarchynullable.models.AFirstDto("copy")
    check(original.copy(id="updated") == org.example.hierarchynullable.models.FirstDto("updated"))'''
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
        check(serializer<org.example.hierarchynullablelimits.models.A47Dto>().descriptor.isNullable)
        check(json.decodeFromString<org.example.hierarchynullablelimits.models.A47Dto>("null") == null)
        val made = org.example.hierarchynullablelimits.models.RootLimitDto("deep")
        val value = org.example.hierarchynullablelimits.models.ConsumerDto(made)
        check(json.decodeFromJsonElement<org.example.hierarchynullablelimits.models.ConsumerDto>(json.encodeToJsonElement(value)) == value)
        check(runCatching { json.decodeFromString<org.example.hierarchynullablelimits.models.ConsumerDto>("{\\"narrowed\\":null}") }.isFailure)
        val wide = json.decodeFromJsonElement<org.example.hierarchynullablelimits.models.WideDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(wide.p0 == null && wide.p239 == null)
    }
    hierarchyNullableAliasContracts()
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runHierarchyNullableAliasContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runHierarchyNullableAliasContracts
    echo "FRESH HIERARCHY NULLABLE ALIAS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh hierarchy nullable alias native projects: $workspace"
