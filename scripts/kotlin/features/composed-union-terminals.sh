#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/product-terminal-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ComposedUnionTerminalsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    if [[ "$style" == plain_class ]]; then flags+=(--deduplicate-models --hoist-shared-shapes --qualify-hoisted-names --merge-identical-models); fi
    cp tests/fixtures/http/composed-union-terminals.json "$project/input.json"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.productterminals --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.productterminals --model-name-suffix Dto "${flags[@]}" --check
    python3 - "$project" <<'PYLIMITS'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);r='#/components/schemas/'
def choice(field,n):return {'oneOf':[{'properties':{field:{'type':'string','const':field+str(i)}}} for i in range(n)]}
s={'Product':{'allOf':[choice('kind',15),choice('mode',16)]},'ProductMaybe':{'$ref':r+'Product','nullable':True},'Small':{'allOf':[choice('kind',2)]},'Wide':{'allOf':[{'properties':{'p'+str(i):{'type':'boolean'} for i in range(239)},'required':['p'+str(i) for i in range(239)]},choice('kind',2)]},'WideMaybe':{'$ref':r+'Wide','nullable':True},'WideFalse':{'$ref':r+'WideMaybe','nullable':False},'Hops':{'properties':{}}}
for i in range(47):s['Hops']['properties']['n'+str(i)]={'$ref':r+('Small' if i==0 else 'Hops/properties/n'+str(i-1))}
s['Deep']={'$ref':r+'Hops/properties/n46'}
s['Typed']={'$ref':r+'Hops/properties/n44','nullable':False}
s['Consumer']={'properties':{'value':{'$ref':r+'Typed','nullable':False}},'required':['value']}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s}}))
PYLIMITS
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.productterminallimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else 'check(org.example.productterminals.models.ProductDto.Note("old",true,"id").copy(body="new").body == "new")'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(r'''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.serializer
fun main()=runBlocking {
    '''+copy+r'''
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val json=Json { encodeDefaults=defaults; explicitNulls=nulls }
        check(serializer<org.example.productterminallimits.models.ProductMaybeDto>().descriptor.isNullable)
        check(!org.example.productterminallimits.models.ProductMaybeDto.serializer().descriptor.isNullable)
        check(json.decodeFromString<org.example.productterminallimits.models.ProductMaybeDto>("null") == null)
        val product=json.decodeFromString<org.example.productterminallimits.models.ProductMaybeDto>("{\"kind\":\"kind14\",\"mode\":\"mode15\"}")
        check(product is org.example.productterminallimits.models.ProductDto.Kind14240)
        val deep: org.example.productterminallimits.models.DeepDto=org.example.productterminallimits.models.SmallDto.Kind0
        val consumer=org.example.productterminallimits.models.ConsumerDto(deep)
        check(json.decodeFromString<org.example.productterminallimits.models.ConsumerDto>(json.encodeToString(consumer)) == consumer)
        check(runCatching { json.decodeFromString<org.example.productterminallimits.models.ConsumerDto>("{\"value\":null}") }.isFailure)
        val wire=JsonObject((0 until 239).associate { "p$it" to JsonPrimitive(true) } + ("kind" to JsonPrimitive("kind0")))
        val wide=json.decodeFromJsonElement<org.example.productterminallimits.models.WideFalseDto>(wire) as org.example.productterminallimits.models.WideDto.Kind0
        check(wide.p0 && wide.p238)
        check(serializer<org.example.productterminallimits.models.WideMaybeDto>().descriptor.isNullable)
        check(!serializer<org.example.productterminallimits.models.WideFalseDto>().descriptor.isNullable)
    }
    composedUnionTerminalsContracts()
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runComposedUnionTerminalsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runComposedUnionTerminalsContracts
    echo "FRESH COMPOSED UNION TERMINALS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh composed union terminal native projects: $workspace"
