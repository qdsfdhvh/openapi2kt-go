#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/wrapper-chain-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ValueClassReferenceChainsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    cp tests/fixtures/http/valueclass-reference-chains.json "$project/input.json"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.wrapperchains --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.wrapperchains --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
s={'Value':{'type':'integer','format':'int64','x-kotlin-value-class':True,'nullable':True,'minimum':1},'NamedAlias':{'$ref':r+'Value'},'Owner':{'properties':{'hop':{'$ref':r+'Value'}}},'Bridge':{'$ref':r+'Owner/properties/hop'},'False':{'$ref':r+'Bridge','nullable':False},'Boxed':{'properties':{'p'+str(i):{'$ref':r+'Bridge'} for i in range(240)},'required':['p'+str(i) for i in range(240)]},'Unboxed':{'properties':{'p'+str(i):{'$ref':r+'False'} for i in range(120)},'required':['p'+str(i) for i in range(120)]},'Hops':{'properties':{}}}
for i in range(47):s['Hops']['properties']['n'+str(i)]={'$ref':r+('Value' if i==0 else 'Hops/properties/n'+str(i-1))}
for i in range(46):s['Hops']['properties']['a'+str(i)]={'$ref':r+('NamedAlias' if i==0 else 'Hops/properties/a'+str(i-1))}
s['DeepNamed']={'$ref':r+'Hops/properties/n46'};s['DeepAlias']={'$ref':r+'Hops/properties/a45'}
s['TypedNamed']={'$ref':r+'Hops/properties/n44','nullable':False};s['TypedAlias']={'$ref':r+'Hops/properties/a43','nullable':False}
s['Consumer']={'properties':{'named':{'$ref':r+'TypedNamed','nullable':False},'alias':{'$ref':r+'TypedAlias','nullable':False}},'required':['named','alias']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.wrapperchainlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else 'check(org.example.wrapperchainlimits.models.ConsumerDto(org.example.wrapperchainlimits.models.ValueDto(1L), org.example.wrapperchainlimits.models.ValueDto(2L)).copy(alias = org.example.wrapperchainlimits.models.ValueDto(3L)).alias.rawValue == 3L)'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(r'''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.serializer
fun main()=runBlocking {
    '''+copy+r'''
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val json=Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(json.decodeFromString<org.example.wrapperchainlimits.models.DeepNamedDto>("null") == null)
        check(json.decodeFromString<org.example.wrapperchainlimits.models.DeepAliasDto>("null") == null)
        val named: org.example.wrapperchainlimits.models.TypedNamedDto=org.example.wrapperchainlimits.models.ValueDto(3L)
        val alias: org.example.wrapperchainlimits.models.TypedAliasDto=org.example.wrapperchainlimits.models.ValueDto(4L)
        val made=org.example.wrapperchainlimits.models.ConsumerDto(named,alias)
        check(json.decodeFromString<org.example.wrapperchainlimits.models.ConsumerDto>(json.encodeToString(made)) == made)
        check(runCatching { json.decodeFromString<org.example.wrapperchainlimits.models.ConsumerDto>("{\"named\":null,\"alias\":null}") }.isFailure)
        val boxed=json.decodeFromJsonElement<org.example.wrapperchainlimits.models.BoxedDto>(JsonObject((0 until 240).associate { "p$it" to JsonNull }))
        check(boxed.p0 == null && boxed.p239 == null)
        val unboxed=json.decodeFromJsonElement<org.example.wrapperchainlimits.models.UnboxedDto>(JsonObject((0 until 120).associate { "p$it" to JsonPrimitive(1L) }))
        check(unboxed.p0.rawValue == 1L && unboxed.p119.rawValue == 1L)
        check(serializer<org.example.wrapperchainlimits.models.BridgeDto>().descriptor.isNullable)
        check(!serializer<org.example.wrapperchainlimits.models.FalseDto>().descriptor.isNullable)
    }
    valueClassReferenceChainsContracts()
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runValueClassReferenceChainsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runValueClassReferenceChainsContracts
    echo "FRESH VALUECLASS REFERENCE CHAINS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh valueclass reference chain native projects: $workspace"
