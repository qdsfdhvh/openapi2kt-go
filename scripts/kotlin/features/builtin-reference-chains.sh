#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/builtin-chain-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/BuiltinReferenceChainsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/builtin-reference-chains.json').read_text())
if checked:
    s=d['components']['schemas']
    s['Text'].update(minLength=2,pattern='^[a-z]+$')
    s['Count'].update(minimum=2,multipleOf=2)
    s['Wide'].update(minimum=2)
    s['Ratio'].update(minimum=0,maximum=10)
    s['Measure'].update(minimum=0,maximum=10)
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
r='#/components/schemas/'
s={'Long':{'type':'integer','format':'int64','nullable':True},'Double':{'type':'number','format':'double','nullable':True},'Text':{'type':'string','nullable':True},'Hops':{'properties':{'long':{'$ref':r+'Long'},'double':{'$ref':r+'Double'},'text':{'$ref':r+'Text'}}},'LongBridge':{'$ref':r+'Hops/properties/long','nullable':False},'DoubleBridge':{'$ref':r+'Hops/properties/double','nullable':False},'BoxedBridge':{'$ref':r+'Hops/properties/long'},'TextBridge':{'$ref':r+'Hops/properties/text','nullable':False}}
if checked:s['Text']['minLength']=2
for name,count,target in [('WideLong',120,'LongBridge'),('WideDouble',120,'DoubleBridge'),('Boxed',240,'BoxedBridge')]:
    s[name]={'properties':{'p'+str(i):{'$ref':r+target} for i in range(count)},'required':['p'+str(i) for i in range(count)]}
for i in range(46):s['Hops']['properties']['n'+str(i)]={'$ref':r+('Long' if i==0 else 'Hops/properties/n'+str(i-1)),'nullable':False}
s['DeepRoot']={'$ref':r+'Hops/properties/n44','nullable':False}
s['DeepConsumer']={'properties':{'cached':{'$ref':r+'DeepRoot','nullable':False},'direct':{'$ref':r+'Hops/properties/n45','nullable':False}},'required':['cached','direct']}
leaf={'$ref':r+'Hops/properties/text','nullable':False}
for _ in range(16):leaf={'type':'array','items':leaf}
s['Checked']={'properties':{'rows':leaf},'required':['rows']}
paths={}
for name,count,target in [('wide',120,'LongBridge'),('boxed',240,'BoxedBridge')]:
    paths['/'+name]={'get':{'parameters':[{'name':'p'+str(i),'in':'query','required':True,'schema':{'$ref':r+target}} for i in range(count)],'responses':{'204':{}}}}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s},'paths':paths},indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.builtinchains --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.builtinchains --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.builtinchainlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else 'check(org.example.builtinchains.models.HolderDto(null,null,null,null,null,null,"ok",null,emptyList(),emptyMap()).copy(narrow="ok").narrow == "ok")'
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
        val deep=org.example.builtinchainlimits.models.DeepConsumerDto(4L,4L)
        check(json.decodeFromString<org.example.builtinchainlimits.models.DeepConsumerDto>(json.encodeToString(deep)) == deep)
        val longs=json.decodeFromJsonElement<org.example.builtinchainlimits.models.WideLongDto>(JsonObject((0 until 120).associate { "p$it" to JsonPrimitive(Long.MAX_VALUE) }))
        check(longs.p0==Long.MAX_VALUE && longs.p119==Long.MAX_VALUE)
        val doubles=json.decodeFromJsonElement<org.example.builtinchainlimits.models.WideDoubleDto>(JsonObject((0 until 120).associate { "p$it" to JsonPrimitive(2.5) }))
        check(doubles.p0==2.5 && doubles.p119==2.5)
        val boxed=json.decodeFromJsonElement<org.example.builtinchainlimits.models.BoxedDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(boxed.p0==null && boxed.p239==null)
        var rows:kotlinx.serialization.json.JsonElement=JsonPrimitive("ok")
        repeat(16) {rows=JsonArray(listOf(rows))}
        val value=json.decodeFromJsonElement<org.example.builtinchainlimits.models.CheckedDto>(JsonObject(mapOf("rows" to rows)))
        check(json.encodeToJsonElement(value)==JsonObject(mapOf("rows" to rows)))
        '''+('''var bad:kotlinx.serialization.json.JsonElement=JsonPrimitive("x")
        repeat(16) {bad=JsonArray(listOf(bad))}
        check(runCatching { json.decodeFromJsonElement<org.example.builtinchainlimits.models.CheckedDto>(JsonObject(mapOf("rows" to bad))) }.isFailure)
        ''' if checked else '')+'''
    }
    builtinReferenceChainsContracts('''+str(checked).lower()+''')
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runBuiltinReferenceChainsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runBuiltinReferenceChainsContracts
    echo "FRESH BUILTIN REFERENCE CHAINS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh builtin reference chain native projects: $workspace"
