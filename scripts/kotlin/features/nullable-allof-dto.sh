#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/nullable-allof-dto-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableAllOfDTOContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/nullable-allof-dtos.json').read_text())
if checked:d['components']['schemas']['Base']['properties']['id']['minLength']=2
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
r='#/components/schemas/'
record={'nullable':True,'allOf':[{'properties':{'id':{'type':'string',**({'minLength':2} if checked else {})}},'required':['id']}]}
s={'ZRecord':record,'Near':{'properties':{'value':{'$ref':r+'A45'}},'required':['value']},'Narrow':{'properties':{'value':{'$ref':r+'A45','nullable':False}},'required':['value']}}
for i in range(46):s[f'A{i:02d}']={'$ref':r+('ZRecord' if i==0 else f'A{i-1:02d}')}
props={f'p{i}':{'type':'boolean'} for i in range(240)}
s['Wide']={'nullable':True,'allOf':[{'properties':props,'required':list(props)}]}
leaf={'type':'string',**({'minLength':1} if checked else {})}
for _ in range(16):leaf={'type':'array','items':leaf}
s['Checked']={'nullable':True,'allOf':[{'properties':{'rows':leaf},'required':['rows']}]}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':s}},indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.nullableallof --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.nullableallof --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.nullablealloflimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);checked=sys.argv[3]=='on'
copy='' if sys.argv[2]=='plain_class' else '''val made=org.example.nullableallof.models.AInheritedDto("ok")
    check(made.copy(id="next").id=="next")'''
copyCheck='' if sys.argv[2]=='plain_class' or not checked else '''try { org.example.nullableallof.models.AInheritedDto("ok").copy(id=""); error("Missing nullable allOf copy validation") }
    catch(cause:IllegalArgumentException) { check(cause.message!!.contains("minLength")) }'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
fun main()=runBlocking {
    '''+copy+'\n'+copyCheck+'''
    val codec=Json
    check(codec.decodeFromString<org.example.nullablealloflimits.models.NearDto>("{\\"value\\":null}").value==null)
    check(codec.decodeFromString<org.example.nullablealloflimits.models.NarrowDto>("{\\"value\\":{\\"id\\":\\"ok\\"}}").value.id=="ok")
    val wideWire=JsonObject((0 until 240).associate { "p$it" to JsonPrimitive(true) })
    val wide:org.example.nullablealloflimits.models.WideDto=codec.decodeFromJsonElement(wideWire)
    check(wide.p0 && wide.p239 && codec.encodeToJsonElement(wide)==wideWire)
    check(codec.decodeFromString<org.example.nullablealloflimits.models.WideDto?>("null")==null)
    var leaf:kotlinx.serialization.json.JsonElement=JsonPrimitive("good")
    repeat(16) { leaf=JsonArray(listOf(leaf)) }
    val deep=JsonObject(mapOf("rows" to leaf))
    val typed:org.example.nullablealloflimits.models.CheckedDto=codec.decodeFromJsonElement(deep)
    check(codec.encodeToJsonElement(typed)==deep)
    nullableAllOfDTOContracts('''+str(checked).lower()+''')
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runNullableAllOfDTOContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNullableAllOfDTOContracts
    echo "FRESH NULLABLE ALLOF DTO PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh nullable allOf DTO native projects: $workspace"
