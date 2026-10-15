#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/borrowed-root-collection-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RootBorrowedCollectionContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/root-borrowed-collection.json').read_text())
if checked:
    c=d['components']['schemas']['Source']['properties'];c['leaf/key~']['minLength']=2;c['list'].update(minItems=1,maxItems=3,uniqueItems=True);c['map'].update(minProperties=1,maxProperties=2);c['mixed'].update(minItems=1,maxItems=2)
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
r='#/components/schemas/'
pfx=r+'Source/properties/'
s={'Source':{'properties':{'text':{'type':['string','null']},'hop':{'$ref':pfx+'text'},'list':{'type':['array','null'],'items':{'$ref':pfx+'hop'}},'map':{'additionalProperties':{'$ref':pfx+'list','nullable':False},'nullable':True}}},'Bridge':{'$ref':pfx+'list'},'FalseBridge':{'$ref':pfx+'list','nullable':False},'MapBridge':{'$ref':pfx+'map','nullable':False}}
if checked:
    z=s['Source']['properties'];z['text']['minLength']=2;z['list'].update(minItems=1,uniqueItems=True);z['map']['minProperties']=1
for name,target in [('Required','FalseBridge'),('Nullable','Bridge'),('MapRequired','MapBridge')]:
    s[name]={'properties':{'p'+str(i):{'$ref':r+target} for i in range(240)},'required':['p'+str(i) for i in range(240)]}
for i in range(43):s['Source']['properties']['n'+str(i)]={'$ref':pfx+('list' if i==0 else 'n'+str(i-1)),'nullable':False}
# The item hop adds two original source refs; reduce incoming hops accordingly.
s['DeepRoot']={'$ref':pfx+'n41','nullable':False}
s['DeepConsumer']={'properties':{'cached':{'$ref':r+'DeepRoot','nullable':False},'direct':{'$ref':pfx+'n42','nullable':False}},'required':['cached','direct']}
leaf={'$ref':r+'FalseBridge'}
for _ in range(15):leaf={'type':'array','items':leaf}
s['Checked']={'properties':{'rows':leaf},'required':['rows']}
paths={}
for name,count,target,stream in [('wide',240,'FalseBridge',False),('mapWide',240,'MapBridge',False),('events',239,'Bridge',True)]:
    response={'200':{'content':{'text/event-stream':{'schema':{'properties':{}}}}}} if stream else {'204':{}}
    paths['/'+name]={'get':{'parameters':[{'name':'p'+str(i),'in':'query','required':True,'schema':{'$ref':r+target}} for i in range(count)],'responses':response}}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s},'paths':paths},indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.borrowedcollectionroots --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.borrowedcollectionroots --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.borrowedcollectionlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);checked=sys.argv[3]=='on'
copy='' if sys.argv[2]=='plain_class' else 'check(org.example.borrowedcollectionroots.models.HolderDto(narrow=listOf("ok"),map=mapOf("key" to listOf("ok")),mixed=listOf(null)).copy(narrow=listOf("future")).narrow==listOf("future"))'
args=', '.join('p'+str(i)+'=wideMap' for i in range(240))
wide='''
    var wideCalls=0
    val wideHttp=io.ktor.client.HttpClient(io.ktor.client.engine.mock.MockEngine { request ->
        wideCalls++; check(request.url.parameters.names()==setOf("key"))
        check(request.url.parameters.getAll("key")==List(240) { "[ok]" })
        respond("",io.ktor.http.HttpStatusCode.NoContent)
    })
    try {
        val wideApi=org.example.borrowedcollectionlimits.api.DefaultApi(wideHttp,"https://example.test/")
        val wideMap=mapOf("key" to listOf("ok"))
        var callback=false
        wideApi.getMapWide('''+args+''',requestBuilder={callback=true})
        check(wideCalls==1 && callback)
'''
modelCheck=''
if checked:
    modelCheck=r'''
        val invalidMapInput=JsonObject((0 until 240).associate { "p$it" to JsonObject(mapOf("key" to JsonArray(listOf(JsonPrimitive(if(it==239) "a" else "ok"))))) })
        try { json.decodeFromJsonElement<org.example.borrowedcollectionlimits.models.MapRequiredDto>(invalidMapInput);error("missing final field map check") }
        catch(failure:Throwable) { check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message?.contains("minLength")==true }) }
'''
    invalid=', '.join('p'+str(i)+'=' + ('mapOf("key" to listOf("a"))' if i==239 else 'wideMap') for i in range(240))
    stream=', '.join('p'+str(i)+'=' + ('emptyList()' if i==238 else 'listOf("ok")') for i in range(239))
    wide+='''
        callback=false
        try { wideApi.getMapWide('''+invalid+''',requestBuilder={callback=true});error("missing wide map check") }
        catch(failure:IllegalArgumentException) { check(failure.message!!.contains("minLength")) }
        check(wideCalls==1 && !callback)
        val cold=wideApi.getEvents('''+stream+''',requestBuilder={callback=true})
        check(wideCalls==1 && !callback)
        try { cold.toList();error("missing cold wide SSE preflight") }
        catch(failure:IllegalArgumentException) { check(failure.message!!.contains("minItems")) }
        check(wideCalls==1 && !callback)
'''
wide+='''
    } finally { wideHttp.close() }
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(r'''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import io.ktor.client.engine.mock.respond
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.flow.toList
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
fun main()=runBlocking {
    '''+copy+wide+r'''
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val json=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val values=listOf(null,"ok")
        val deep=org.example.borrowedcollectionlimits.models.DeepConsumerDto(values,values)
        check(json.encodeToJsonElement(json.decodeFromString<org.example.borrowedcollectionlimits.models.DeepConsumerDto>(json.encodeToString(deep))) == json.encodeToJsonElement(deep))
        val input=JsonObject((0 until 240).associate { "p$it" to JsonArray(listOf(JsonNull,JsonPrimitive("ok"))) })
        val required=json.decodeFromJsonElement<org.example.borrowedcollectionlimits.models.RequiredDto>(input)
        check(required.p0==values && required.p239==values)
        val nullable=json.decodeFromJsonElement<org.example.borrowedcollectionlimits.models.NullableDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(nullable.p0==null && nullable.p239==null)
        val map=json.decodeFromJsonElement<org.example.borrowedcollectionlimits.models.MapRequiredDto>(JsonObject((0 until 240).associate { "p$it" to JsonObject(mapOf("key" to JsonArray(listOf(JsonNull,JsonPrimitive("ok"))))) }))
        check(map.p0==mapOf("key" to values) && map.p239==map.p0)
'''+modelCheck+r'''
        val checkedText="[".repeat(15)+"[null,\"ok\"]"+"]".repeat(15)
        val checked=json.decodeFromString<org.example.borrowedcollectionlimits.models.CheckedDto>("{\"rows\":$checkedText}")
        check(json.encodeToJsonElement(checked)==json.parseToJsonElement("{\"rows\":$checkedText}"))
    }
    rootBorrowedCollectionContracts('''+str(checked).lower()+r''')
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runRootBorrowedCollectionContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runRootBorrowedCollectionContracts
    echo "FRESH ROOT BORROWED COLLECTION PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh borrowed root collection native projects: $workspace"
