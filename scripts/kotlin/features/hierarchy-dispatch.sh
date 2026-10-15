#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/hierarchy-dispatch-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/HierarchyDispatchContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    "$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-dispatch.json --output "$project/build/generated" --package-name org.example.hierarchydispatch --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input tests/fixtures/http/hierarchy-dispatch.json --output "$project/build/generated" --package-name org.example.hierarchydispatch --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
def ref(name):return {'$ref':r+name}
def union(names,field=None,mapping=None):
 v={'oneOf':[ref(name) for name in names]}
 if field is not None:v['discriminator']={'propertyName':field,'mapping':mapping or {name.lower():name for name in names}}
 return v
schemas={
 'WideRoot':union(['WideGroup','Other'],'family',{'group':'WideGroup','other':'Other'}),
 'WideGroup':union(['WideFirst','WideSecond']),
 'Other':{'properties':{'family':{'type':'string','const':'other'}}},
 'VocabularyRoot':union(['VocabularyGroup','VocabularyOther'],'family',{'group':'VocabularyGroup','other':'VocabularyOther'}),
 'VocabularyGroup':union(['VocabularyFirst','VocabularySecond']),
 'VocabularyOther':{'properties':{'family':{'type':'string','const':'other'}}},
}
for index,name in enumerate(['WideFirst','WideSecond']):
 properties={'family':{'type':'string','const':'group'},'tag':{'type':'string','const':'t'+str(index)},'mode':{'type':'string','const':'m'+str(index)}}
 properties.update({'p'+str(i):{'type':'boolean'} for i in range(237)})
 schemas[name]={'properties':properties,'required':['p'+str(i) for i in range(237)]}
for index,name in enumerate(['VocabularyFirst','VocabularySecond']):
 schemas[name]={'properties':{'family':{'type':'string','const':'group'},'tag':{'enum':['v'+str(i) for i in range(index*120,(index+1)*120)]},'mode':{'type':'string','const':'m'+str(index)},'id':{'type':'string'}},'required':['id']}
# 48 graph levels reuse one actual ancestor enum property. Nearest leaf mappings
# differ; the ancestor route keeps its own selected wire behavior.
for i in range(46):schemas['D'+str(i)]=union(['D'+str(i+1)],'tag',{'one':'D'+str(i+1)})
schemas['D46']=union(['DeepFirst','DeepSecond'],'tag',{'one':'DeepFirst','two':'DeepSecond'})
schemas['DeepFirst']={'properties':{'id':{'type':'string'}},'required':['id']}
schemas['DeepSecond']={'properties':{'id':{'type':'string'}},'required':['id']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.hierarchydispatchlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else '''val original = org.example.hierarchydispatch.models.RangeDto(org.example.hierarchydispatch.models.GroupDtoTag.THREE,org.example.hierarchydispatch.models.GroupDtoMode.SLOW,"copy")
    check(original.copy(id="updated").tag === original.tag)'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.decodeFromJsonElement
fun main()=runBlocking {
    '''+copy+'''
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val json = Json { encodeDefaults=defaults; explicitNulls=nulls }
        val first: org.example.hierarchydispatchlimits.models.D0Dto = org.example.hierarchydispatchlimits.models.DeepFirstDto("deep")
        val wire = json.encodeToJsonElement(first)
        check(json.decodeFromJsonElement<org.example.hierarchydispatchlimits.models.D0Dto>(wire) == first)
        val second: org.example.hierarchydispatchlimits.models.D46Dto = org.example.hierarchydispatchlimits.models.DeepSecondDto("inner")
        val inner = json.encodeToJsonElement(second)
        check(json.decodeFromJsonElement<org.example.hierarchydispatchlimits.models.D46Dto>(inner) == second)
        check(json.decodeFromJsonElement<org.example.hierarchydispatchlimits.models.D0Dto>(inner) is org.example.hierarchydispatchlimits.models.UnknownD0Dto)
        check(org.example.hierarchydispatchlimits.models.VocabularyGroupDtoTag.entries.size==240)
        val vocabulary = org.example.hierarchydispatchlimits.models.VocabularySecondDto(org.example.hierarchydispatchlimits.models.VocabularyGroupDtoTag.fromSerialName("v239"),"enum")
        check(json.decodeFromJsonElement<org.example.hierarchydispatchlimits.models.VocabularyRootDto>(json.encodeToJsonElement<org.example.hierarchydispatchlimits.models.VocabularyRootDto>(vocabulary)) == vocabulary)
    }
    hierarchyDispatchContracts()
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runHierarchyDispatchContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runHierarchyDispatchContracts
    echo "FRESH HIERARCHY DISPATCH PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh hierarchy dispatch native projects: $workspace"
