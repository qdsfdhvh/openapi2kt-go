#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/recursive-cover-chain-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RecursiveCoverReferenceChainsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    python3 - "$project" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1])
d=json.loads(Path('tests/fixtures/http/recursive-cover-reference-chains.json').read_text())
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
r='#/components/schemas/'
def cover(array,object):return {'oneOf':[{'type':'null'},{'type':'string'},{'type':'boolean'},{'type':'number'},{'type':'array','items':{'$ref':r+array}},{'type':'object','additionalProperties':{'$ref':r+object}}]}
s={'Pair':cover('ToOther','Other'),'Other':cover('ToPair','Pair'),'ToOther':{'$ref':r+'Other'},'ToPair':{'$ref':r+'Pair'},'Self':cover('Self','Self'),'Hops':{'properties':{'pair':{'$ref':r+'Pair','nullable':True},'self':{'$ref':r+'Self','nullable':True}}},'PairBridge':{'$ref':r+'Hops/properties/pair'},'SelfBridge':{'$ref':r+'Hops/properties/self'},'PairFalse':{'$ref':r+'Hops/properties/pair','nullable':False},'SelfFalse':{'$ref':r+'Hops/properties/self','nullable':False}}
for name,target in [('PairRequired','PairFalse'),('PairNullable','PairBridge'),('SelfRequired','SelfFalse'),('SelfNullable','SelfBridge')]:
    s[name]={'properties':{'p'+str(i):{'$ref':r+target} for i in range(240)},'required':['p'+str(i) for i in range(240)]}
for prefix,target,count in [('n','Pair',40),('self','Self',44)]:
    for i in range(count):s['Hops']['properties'][prefix+str(i)]={'$ref':r+(target if i==0 else 'Hops/properties/'+prefix+str(i-1)),'nullable':False}
s['DeepRoot']={'$ref':r+'Hops/properties/n38','nullable':False}
s['SelfDeepRoot']={'$ref':r+'Hops/properties/self42','nullable':False}
s['DeepConsumer']={'properties':{'cached':{'$ref':r+'DeepRoot','nullable':False},'direct':{'$ref':r+'Hops/properties/n39','nullable':False}},'required':['cached','direct']}
s['SelfDeepConsumer']={'properties':{'cached':{'$ref':r+'SelfDeepRoot','nullable':False},'direct':{'$ref':r+'Hops/properties/self43','nullable':False}},'required':['cached','direct']}
paths={}
for name,count,target,stream in [('widePair',240,'PairFalse',False),('wideSelf',240,'SelfFalse',False),('eventsPair',239,'PairBridge',True),('eventsSelf',239,'SelfBridge',True)]:
    response={'200':{'content':{'text/event-stream':{'schema':{'properties':{}}}}}} if stream else {'204':{}}
    paths['/'+name]={'get':{'parameters':[{'name':'p'+str(i),'in':'query','required':True,'schema':{'$ref':r+target}} for i in range(count)],'responses':response}}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s},'paths':paths},indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.coverchains --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.coverchains --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.coverchainlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1])
copy='' if sys.argv[2]=='plain_class' else '''check(org.example.coverchains.models.HolderDto(null,null,JsonNull,null,JsonNull,emptyList(),JsonObject(emptyMap()),null,null,JsonNull).copy(forcedFalse=JsonPrimitive("future")).forcedFalse == JsonPrimitive("future"))'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
fun main()=runBlocking {
    '''+copy+'''
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val json=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val deep=org.example.coverchainlimits.models.DeepConsumerDto(JsonNull,JsonPrimitive("deep"))
        check(json.encodeToJsonElement(json.decodeFromString<org.example.coverchainlimits.models.DeepConsumerDto>(json.encodeToString(deep))) == json.encodeToJsonElement(deep))
        val selfDeep=org.example.coverchainlimits.models.SelfDeepConsumerDto(JsonNull,JsonPrimitive("self"))
        check(json.encodeToJsonElement(json.decodeFromString<org.example.coverchainlimits.models.SelfDeepConsumerDto>(json.encodeToString(selfDeep))) == json.encodeToJsonElement(selfDeep))
        val input=JsonObject((0 until 240).associate { "p$it" to JsonNull })
        val raw=json.decodeFromJsonElement<org.example.coverchainlimits.models.PairRequiredDto>(input)
        val any=json.decodeFromJsonElement<org.example.coverchainlimits.models.SelfRequiredDto>(input)
        val rawNull=json.decodeFromJsonElement<org.example.coverchainlimits.models.PairNullableDto>(input)
        val anyNull=json.decodeFromJsonElement<org.example.coverchainlimits.models.SelfNullableDto>(input)
        check(raw.p0===JsonNull && raw.p239===JsonNull && any.p0===JsonNull && any.p239===JsonNull)
        check(rawNull.p0==null && rawNull.p239==null && anyNull.p0==null && anyNull.p239==null)
    }
    recursiveCoverReferenceChainsContracts()
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runRecursiveCoverReferenceChainsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runRecursiveCoverReferenceChainsContracts
    echo "FRESH RECURSIVE COVER REFERENCE CHAINS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh recursive cover reference chain native projects: $workspace"
