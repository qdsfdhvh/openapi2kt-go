#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/bytes-chain-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BytesReferenceChainsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/bytes-reference-chains.json').read_text())
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
r='#/components/schemas/'
s={'Value':{'type':'string','format':'byte','nullable':True},'Hops':{'properties':{'value':{'$ref':r+'Value'}}},'Bridge':{'$ref':r+'Hops/properties/value'},'NarrowBridge':{'$ref':r+'Hops/properties/value','nullable':False}}
for name,count,target in [('Required',240,'NarrowBridge'),('Boxed',240,'Bridge')]:
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
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.byteschains --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.byteschains --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.byteschainlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else 'check(org.example.byteschains.models.HolderDto(null,byteArrayOf(1),byteArrayOf(1),null,byteArrayOf(1),emptyList(),emptyMap(),null,byteArrayOf(1),byteArrayOf(1),byteArrayOf(1),null,byteArrayOf(1),null,byteArrayOf(1),null,byteArrayOf(1)).copy(forcedFalse=byteArrayOf(2)).forcedFalse.contentEquals(byteArrayOf(2)))'
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
        val deep=org.example.byteschainlimits.models.DeepConsumerDto(byteArrayOf(4),byteArrayOf(4))
        check(json.encodeToJsonElement(json.decodeFromString<org.example.byteschainlimits.models.DeepConsumerDto>(json.encodeToString(deep))) == json.encodeToJsonElement(deep))
        val required=json.decodeFromJsonElement<org.example.byteschainlimits.models.RequiredDto>(JsonObject((0 until 240).associate { "p$it" to JsonArray(listOf(JsonPrimitive(1))) }))
        check(required.p0.contentEquals(byteArrayOf(1)) && required.p239.contentEquals(byteArrayOf(1)))
        val boxed=json.decodeFromJsonElement<org.example.byteschainlimits.models.BoxedDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(boxed.p0==null && boxed.p239==null)
    }
    bytesReferenceChainsContracts()
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runBytesReferenceChainsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runBytesReferenceChainsContracts
    echo "FRESH BYTES REFERENCE CHAINS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh byte/binary reference chain native projects: $workspace"
