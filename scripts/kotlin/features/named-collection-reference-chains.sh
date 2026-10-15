#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/named-collection-chain-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NamedCollectionReferenceChainsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/named-collection-reference-chains.json').read_text())
if checked:
    c=d['components']['schemas'];c['ZValues'].update(minItems=1,maxItems=3,uniqueItems=True);c['ZValues']['items']['minLength']=2;c['ZMap']['minProperties']=1;c['ZMap']['additionalProperties']['minLength']=2;c['ZItems']['items']['minLength']=2;c['ZRows']['items']['properties']['id']['minLength']=1;c['ZOpen']['minProperties']=1
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
r='#/components/schemas/'
s={'Value':{'type':'array','nullable':True,'items':{'type':['string','null']}},'Mapping':{'additionalProperties':{'type':['string','null']},'nullable':True},'Open':{'additionalProperties':True,'nullable':True},'Rows':{'type':'array','items':{'properties':{'id':{'type':'string'}},'required':['id']}},'Hops':{'properties':{'value':{'$ref':r+'Value'},'map':{'$ref':r+'Mapping'},'open':{'$ref':r+'Open'},'rows':{'$ref':r+'Rows'}}},'Bridge':{'$ref':r+'Hops/properties/value'},'FalseBridge':{'$ref':r+'Hops/properties/value','nullable':False},'MapBridge':{'$ref':r+'Hops/properties/map','nullable':False},'OpenBridge':{'$ref':r+'Hops/properties/open'},'RowsBridge':{'$ref':r+'Hops/properties/rows'}}
if checked:
    s['Value'].update(minItems=1,uniqueItems=True);s['Value']['items']['minLength']=2;s['Mapping']['minProperties']=1;s['Mapping']['additionalProperties']['minLength']=2;s['Open']['minProperties']=1;s['Rows']['items']['properties']['id']['minLength']=1
for name,target in [('Required','FalseBridge'),('Nullable','Bridge'),('MapRequired','MapBridge'),('RowsRequired','RowsBridge')]:
    s[name]={'properties':{'p'+str(i):{'$ref':r+target} for i in range(240)},'required':['p'+str(i) for i in range(240)]}
for i in range(45):s['Hops']['properties']['n'+str(i)]={'$ref':r+('Value' if i==0 else 'Hops/properties/n'+str(i-1)),'nullable':False}
s['DeepRoot']={'$ref':r+'Hops/properties/n43','nullable':False}
s['DeepConsumer']={'properties':{'cached':{'$ref':r+'DeepRoot','nullable':False},'direct':{'$ref':r+'Hops/properties/n44','nullable':False}},'required':['cached','direct']}
leaf={'$ref':r+'Hops/properties/value','nullable':False}
for _ in range(15):leaf={'type':'array','items':leaf}
s['Checked']={'properties':{'rows':leaf},'required':['rows']}
paths={}
for name,count,target,stream in [('wide',240,'FalseBridge',False),('mapWide',240,'MapBridge',False),('events',239,'Bridge',True)]:
    response={'200':{'content':{'text/event-stream':{'schema':{'properties':{}}}}}} if stream else {'204':{}}
    paths['/'+name]={'get':{'parameters':[{'name':'p'+str(i),'in':'query','required':True,'schema':{'$ref':r+target}} for i in range(count)],'responses':response}}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s},'paths':paths},indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.namedcollectionchains --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.namedcollectionchains --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.collectionchainlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);checked=sys.argv[3]=='on'
copy='' if sys.argv[2]=='plain_class' else 'check(org.example.namedcollectionchains.models.HolderDto(narrow=listOf("ok"),wrappedNarrow=listOf("ok"),mapNarrow=mapOf("key" to "ok"),mapUseNarrow=mapOf("key" to "ok"),rows=emptyList(),nested=emptyList()).copy(narrow=listOf("future")).narrow==listOf("future"))'
args=', '.join('p'+str(i)+'=wideMap' for i in range(240))
wide='''
    var wideCalls=0
    val wideHttp=io.ktor.client.HttpClient(io.ktor.client.engine.mock.MockEngine { request ->
        wideCalls++; check(request.url.parameters.names().size==240)
        respond("",io.ktor.http.HttpStatusCode.NoContent)
    })
    try {
        val wideApi=org.example.collectionchainlimits.api.DefaultApi(wideHttp,"https://example.test/")
        val wideMap=mapOf("key" to "ok")
        var callback=false
        wideApi.getMapWide('''+args+''',requestBuilder={callback=true})
        check(wideCalls==1 && callback)
'''
modelCheck=''
if checked:
    modelCheck=r'''
        val invalidMapInput=JsonObject((0 until 240).associate { "p$it" to JsonObject(mapOf("key" to JsonPrimitive(if(it==239) "a" else "ok"))) })
        try { json.decodeFromJsonElement<org.example.collectionchainlimits.models.MapRequiredDto>(invalidMapInput);error("missing final field map check") }
        catch(failure:Throwable) { check(generateSequence(failure) { it.cause }.any { it is IllegalArgumentException && it.message?.contains("minLength")==true }) }
'''
    invalid=', '.join('p'+str(i)+'=' + ('mapOf("key" to "a")' if i==239 else 'wideMap') for i in range(240))
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
        val deep=org.example.collectionchainlimits.models.DeepConsumerDto(values,values)
        check(json.encodeToJsonElement(json.decodeFromString<org.example.collectionchainlimits.models.DeepConsumerDto>(json.encodeToString(deep))) == json.encodeToJsonElement(deep))
        val input=JsonObject((0 until 240).associate { "p$it" to JsonArray(listOf(JsonNull,JsonPrimitive("ok"))) })
        val required=json.decodeFromJsonElement<org.example.collectionchainlimits.models.RequiredDto>(input)
        check(required.p0==values && required.p239==values)
        val nullable=json.decodeFromJsonElement<org.example.collectionchainlimits.models.NullableDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(nullable.p0==null && nullable.p239==null)
        val map=json.decodeFromJsonElement<org.example.collectionchainlimits.models.MapRequiredDto>(JsonObject((0 until 240).associate { "p$it" to JsonObject(mapOf("key" to JsonPrimitive("ok"))) }))
        check(map.p0==mapOf("key" to "ok") && map.p239==map.p0)
'''+modelCheck+r'''
        val rows=json.decodeFromJsonElement<org.example.collectionchainlimits.models.RowsRequiredDto>(JsonObject((0 until 240).associate { "p$it" to JsonArray(listOf(JsonObject(mapOf("id" to JsonPrimitive("ok"))))) }))
        check(rows.p0.first().id=="ok" && rows.p239.first().id=="ok")
        val checkedText="[".repeat(15)+"[null,\"ok\"]"+"]".repeat(15)
        val checked=json.decodeFromString<org.example.collectionchainlimits.models.CheckedDto>("{\"rows\":$checkedText}")
        check(json.encodeToJsonElement(checked)==json.parseToJsonElement("{\"rows\":$checkedText}"))
    }
    namedCollectionReferenceChainsContracts('''+str(checked).lower()+r''')
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runNamedCollectionReferenceChainsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNamedCollectionReferenceChainsContracts
    echo "FRESH NAMED COLLECTION REFERENCE CHAINS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh named collection reference chain native projects: $workspace"
