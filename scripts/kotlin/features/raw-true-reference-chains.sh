#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/raw-true-chain-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/RawTrueReferenceChainsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    python3 - "$project" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1])
d=json.loads(Path('tests/fixtures/http/raw-true-reference-chains.json').read_text())
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
r='#/components/schemas/'
s={'Raw':{'type':'object','nullable':True},'Any':True,'EmptyProduct':{'allOf':[{'type':'object'},{'properties':{}}]},'Hops':{'properties':{'raw':{'$ref':r+'Raw'},'any':{'$ref':r+'Any','nullable':True},'empty':{'$ref':r+'EmptyProduct'}}},'RawBridge':{'$ref':r+'Hops/properties/raw'},'AnyBridge':{'$ref':r+'Hops/properties/any'},'NarrowBridge':{'$ref':r+'Hops/properties/raw','nullable':False},'AnyFalseBridge':{'$ref':r+'Hops/properties/any','nullable':False},'EmptyBridge':{'$ref':r+'Hops/properties/empty'}}
for name,target in [('RawRequired','NarrowBridge'),('RawNullable','RawBridge'),('TrueRequired','AnyFalseBridge'),('TrueNullable','AnyBridge')]:
    s[name]={'properties':{'p'+str(i):{'$ref':r+target} for i in range(240)},'required':['p'+str(i) for i in range(240)]}
for i in range(46):s['Hops']['properties']['n'+str(i)]={'$ref':r+('Raw' if i==0 else 'Hops/properties/n'+str(i-1)),'nullable':False}
s['DeepRoot']={'$ref':r+'Hops/properties/n44','nullable':False}
s['DeepConsumer']={'properties':{'cached':{'$ref':r+'DeepRoot','nullable':False},'direct':{'$ref':r+'Hops/properties/n45','nullable':False}},'required':['cached','direct']}
paths={}
for name,count,target,stream in [('wideRaw',240,'NarrowBridge',False),('wideTrue',240,'AnyFalseBridge',False),('eventsRaw',239,'RawBridge',True),('eventsTrue',239,'AnyBridge',True)]:
    response={'200':{'content':{'text/event-stream':{'schema':{'properties':{}}}}}} if stream else {'204':{}}
    paths['/'+name]={'get':{'parameters':[{'name':'p'+str(i),'in':'query','required':True,'schema':{'$ref':r+target}} for i in range(count)],'responses':response}}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s},'paths':paths},indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.rawtruechains --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.rawtruechains --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.rawtruechainlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1])
copy='' if sys.argv[2]=='plain_class' else '''check(org.example.rawtruechains.models.HolderDto(null,null,JsonNull,null,JsonNull,emptyList(),JsonObject(emptyMap()),null,null,JsonNull).copy(forcedFalse=JsonPrimitive("future")).forcedFalse == JsonPrimitive("future"))'''
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
    val empty: org.example.rawtruechainlimits.models.EmptyBridgeDto=JsonPrimitive("not an object")
    check(empty==JsonPrimitive("not an object"))
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val json=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val deep=org.example.rawtruechainlimits.models.DeepConsumerDto(JsonNull,JsonPrimitive("deep"))
        check(json.encodeToJsonElement(json.decodeFromString<org.example.rawtruechainlimits.models.DeepConsumerDto>(json.encodeToString(deep))) == json.encodeToJsonElement(deep))
        val input=JsonObject((0 until 240).associate { "p$it" to JsonNull })
        val raw=json.decodeFromJsonElement<org.example.rawtruechainlimits.models.RawRequiredDto>(input)
        val any=json.decodeFromJsonElement<org.example.rawtruechainlimits.models.TrueRequiredDto>(input)
        val rawNull=json.decodeFromJsonElement<org.example.rawtruechainlimits.models.RawNullableDto>(input)
        val anyNull=json.decodeFromJsonElement<org.example.rawtruechainlimits.models.TrueNullableDto>(input)
        check(raw.p0===JsonNull && raw.p239===JsonNull && any.p0===JsonNull && any.p239===JsonNull)
        check(rawNull.p0==null && rawNull.p239==null && anyNull.p0==null && anyNull.p239==null)
    }
    rawTrueReferenceChainsContracts()
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runRawTrueReferenceChainsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runRawTrueReferenceChainsContracts
    echo "FRESH RAW TRUE REFERENCE CHAINS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh raw true reference chain native projects: $workspace"
