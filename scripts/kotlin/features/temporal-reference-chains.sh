#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/temporal-chain-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TemporalReferenceChainsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    cp tests/fixtures/http/temporal-reference-chains.json "$project/input.json"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.temporalchains --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.temporalchains --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
s={'Time':{'type':'string','format':'date-time','nullable':True},'TimeAlias':{'$ref':r+'Time'},'Owner':{'properties':{'hop':{'$ref':r+'Time'}}},'Bridge':{'$ref':r+'Owner/properties/hop'},'False':{'$ref':r+'Bridge','nullable':False},'Wide':{'properties':{'p'+str(i):{'$ref':r+'False'} for i in range(240)},'required':['p'+str(i) for i in range(240)]},'Hops':{'properties':{}}}
for i in range(47):s['Hops']['properties']['n'+str(i)]={'$ref':r+('Time' if i==0 else 'Hops/properties/n'+str(i-1))}
for i in range(46):s['Hops']['properties']['a'+str(i)]={'$ref':r+('TimeAlias' if i==0 else 'Hops/properties/a'+str(i-1))}
s['DeepNamed']={'$ref':r+'Hops/properties/n46'};s['DeepAlias']={'$ref':r+'Hops/properties/a45'}
s['TypedNamed']={'$ref':r+'Hops/properties/n44','nullable':False};s['TypedAlias']={'$ref':r+'Hops/properties/a43','nullable':False}
s['Consumer']={'properties':{'named':{'$ref':r+'TypedNamed','nullable':False},'alias':{'$ref':r+'TypedAlias','nullable':False}},'required':['named','alias']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.temporalchainlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else 'check(org.example.temporalchains.models.HolderDto(null,stamp,null,stamp,emptyList(),emptyMap()).copy(stamp=stamp).stamp == stamp)'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(r'''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class, kotlin.time.ExperimentalTime::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.serializer
fun main()=runBlocking {
    val stamp=org.example.temporalchainlimits.models.FalseDto.parse("2024-02-29T12:34:56Z")
    '''+copy+r'''
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val json=Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(json.decodeFromString<org.example.temporalchainlimits.models.DeepNamedDto>("null") == null)
        check(json.decodeFromString<org.example.temporalchainlimits.models.DeepAliasDto>("null") == null)
        check(serializer<org.example.temporalchainlimits.models.BridgeDto>().descriptor.isNullable)
        check(!serializer<org.example.temporalchainlimits.models.FalseDto>().descriptor.isNullable)
        val named:org.example.temporalchainlimits.models.TypedNamedDto=stamp
        val alias:org.example.temporalchainlimits.models.TypedAliasDto=stamp
        val consumer=org.example.temporalchainlimits.models.ConsumerDto(named,alias)
        check(json.decodeFromString<org.example.temporalchainlimits.models.ConsumerDto>(json.encodeToString(consumer)) == consumer)
        check(runCatching { json.decodeFromString<org.example.temporalchainlimits.models.ConsumerDto>("{\"named\":null,\"alias\":null}") }.isFailure)
        val wide=json.decodeFromJsonElement<org.example.temporalchainlimits.models.WideDto>(JsonObject((0 until 240).associate { "p$it" to JsonPrimitive(stamp.toString()) }))
        check(wide.p0 == stamp && wide.p239 == stamp)
    }
    temporalReferenceChainsContracts()
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runTemporalReferenceChainsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTemporalReferenceChainsContracts
    echo "FRESH TEMPORAL REFERENCE CHAINS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh temporal reference chain native projects: $workspace"
