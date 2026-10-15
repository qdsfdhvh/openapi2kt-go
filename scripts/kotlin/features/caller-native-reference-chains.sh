#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/caller-native-chain-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/CallerNativeReferenceChainsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    mkdir -p "$project/src/main/kotlin/org/example/scalarvalues"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/scalarvalues/Money.kt "$project/src/main/kotlin/org/example/scalarvalues/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/caller-native-reference-chains.json').read_text())
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
r='#/components/schemas/'
s={'Value':{'x-native-type':'org.example.scalarvalues.Money','nullable':True},'Long':{'x-native-type':'kotlin.Long','nullable':True},'Double':{'x-native-type':'kotlin.Double','nullable':True},'Hops':{'properties':{'value':{'$ref':r+'Value'}}},'Bridge':{'$ref':r+'Hops/properties/value'},'NarrowBridge':{'$ref':r+'Hops/properties/value','nullable':False}}
s['Hops']['properties']['long']={'$ref':r+'Long'}
s['Hops']['properties']['double']={'$ref':r+'Double'}
s['LongBridge']={'$ref':r+'Hops/properties/long','nullable':False}
s['DoubleBridge']={'$ref':r+'Hops/properties/double','nullable':False}
for name,count,target in [('Required',240,'NarrowBridge'),('Boxed',240,'Bridge'),('LongRequired',120,'LongBridge'),('DoubleRequired',120,'DoubleBridge')]:
    s[name]={'properties':{'p'+str(i):{'$ref':r+target} for i in range(count)},'required':['p'+str(i) for i in range(count)]}
for i in range(46):s['Hops']['properties']['n'+str(i)]={'$ref':r+('Value' if i==0 else 'Hops/properties/n'+str(i-1)),'nullable':False}
s['DeepRoot']={'$ref':r+'Hops/properties/n44','nullable':False}
s['DeepConsumer']={'properties':{'cached':{'$ref':r+'DeepRoot','nullable':False},'direct':{'$ref':r+'Hops/properties/n45','nullable':False}},'required':['cached','direct']}
paths={}
for name,count,target,stream in [('wide',240,'NarrowBridge',False),('boxed',240,'Bridge',False),('eventsWide',239,'NarrowBridge',True),('eventsBoxed',239,'Bridge',True)]:
    response={'200':{'content':{'text/event-stream':{'schema':{'properties':{}}}}}} if stream else {'204':{}}
    paths['/'+name]={'get':{'parameters':[{'name':'p'+str(i),'in':'query','required':True,'schema':{'$ref':r+target}} for i in range(count)],'responses':response}}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s},'paths':paths},indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.nativechains --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.nativechains --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.nativechainlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else 'check(org.example.nativechains.models.HolderDto(inherited=null,forcedTrue=null,forcedFalse=org.example.scalarvalues.Money("ok"),wrapper=null,wrapperFalse=org.example.scalarvalues.Money("ok"),items=emptyList(),values=emptyMap(),requiredNull=null,directMoney=null,directFalse=org.example.scalarvalues.Money("ok"),directWrapped=null).copy(forcedFalse=org.example.scalarvalues.Money("future")).forcedFalse.decimal == "future")'
checked=sys.argv[3]=='on'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
fun main()=runBlocking {
    '''+copy+'''
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val json=Json { encodeDefaults=defaults; explicitNulls=nulls }
        val deep=org.example.nativechainlimits.models.DeepConsumerDto(org.example.scalarvalues.Money("ok"),org.example.scalarvalues.Money("ok"))
        check(json.encodeToJsonElement(json.decodeFromString<org.example.nativechainlimits.models.DeepConsumerDto>(json.encodeToString(deep))) == json.encodeToJsonElement(deep))
        val required=json.decodeFromJsonElement<org.example.nativechainlimits.models.RequiredDto>(JsonObject((0 until 240).associate { "p$it" to JsonPrimitive("ok") }))
        check(required.p0.decimal=="ok" && required.p239.decimal=="ok")
        val longs=json.decodeFromJsonElement<org.example.nativechainlimits.models.LongRequiredDto>(JsonObject((0 until 120).associate {"p$it" to JsonPrimitive(Long.MAX_VALUE)}))
        check(longs.p0==Long.MAX_VALUE && longs.p119==Long.MAX_VALUE)
        val doubles=json.decodeFromJsonElement<org.example.nativechainlimits.models.DoubleRequiredDto>(JsonObject((0 until 120).associate {"p$it" to JsonPrimitive(2.5)}))
        check(doubles.p0==2.5 && doubles.p119==2.5)
        val boxed=json.decodeFromJsonElement<org.example.nativechainlimits.models.BoxedDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(boxed.p0==null && boxed.p239==null)
    }
    callerNativeReferenceChainsContracts()
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runCallerNativeReferenceChainsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runCallerNativeReferenceChainsContracts
    echo "FRESH CALLER NATIVE REFERENCE CHAINS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh caller-native reference chain native projects: $workspace"
