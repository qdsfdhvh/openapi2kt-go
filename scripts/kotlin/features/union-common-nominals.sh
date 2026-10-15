#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/union-common-nominal-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnionCommonNominalsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/union-common-nominals.json').read_text())
if checked:d['components']['schemas']['Record']['properties']['id']['minLength']=2
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
r='#/components/schemas/'
def choice(props):return {'oneOf':[{'properties':{'kind':{'const':tag},**props},'required':list(props)} for tag in ('a','b')]}
leaf={'type':'string',**({'minLength':1} if checked else {})}
for _ in range(16):leaf={'type':'array','items':leaf}
s={'Record':{'properties':{'id':{'type':'string'}},'required':['id']},'CheckedValue':{'properties':{'rows':leaf},'required':['rows']},'Wide':choice({f'p{i}':{'$ref':r+'Record'} for i in range(239)}),'Checked':choice({'value':{'$ref':r+'CheckedValue'}}),'Hops':{'properties':{}}}
for i in range(45):s['Hops']['properties'][f'h{i}']={'$ref':r+('Record' if i==0 else f'Hops/properties/h{i-1}'),'nullable':False}
s['Near']=choice({'value':{'$ref':r+'Hops/properties/h44','nullable':False}})
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s}},indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.commonnominals --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.commonnominals --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.commonnominallimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);checked=sys.argv[3]=='on'
copy='' if sys.argv[2]=='plain_class' else '''val value=org.example.commonnominals.models.RecordDto("ok")
    check(org.example.commonnominals.models.DirectDto.A(value).copy(value=org.example.commonnominals.models.RecordDto("next")).value.id=="next")'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
fun main()=runBlocking {
    '''+copy+'''
    val codec=Json
    val input=JsonObject(mapOf("kind" to JsonPrimitive("a"))+(0 until 239).associate { "p$it" to JsonObject(mapOf("id" to JsonPrimitive("value"))) })
    val wide:org.example.commonnominallimits.models.WideDto=codec.decodeFromJsonElement(input)
    check(wide.p0.id=="value" && wide.p238.id=="value" && codec.encodeToJsonElement(wide)==input)
    val near:org.example.commonnominallimits.models.NearDto=org.example.commonnominallimits.models.NearDto.B(org.example.commonnominallimits.models.RecordDto("near"))
    check(near.value.id=="near")
    var leaf:kotlinx.serialization.json.JsonElement=JsonPrimitive("good")
    repeat(16) { leaf=JsonArray(listOf(leaf)) }
    val deep=JsonObject(mapOf("kind" to JsonPrimitive("a"),"value" to JsonObject(mapOf("rows" to leaf))))
    val typed:org.example.commonnominallimits.models.CheckedDto=codec.decodeFromJsonElement(deep)
    check(codec.encodeToJsonElement(typed)==deep)
    unionCommonNominalsContracts('''+str(checked).lower()+''')
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runUnionCommonNominalsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runUnionCommonNominalsContracts
    echo "FRESH UNION COMMON NOMINALS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh common nominal native projects: $workspace"
