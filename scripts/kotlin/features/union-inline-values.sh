#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/union-inline-value-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnionInlineValuesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/union-inline-values.json').read_text())
def add_checks(v):
    if isinstance(v,dict):
        if v.get('type')=='string':v['minLength']=2
        for value in list(v.values()):add_checks(value)
    elif isinstance(v,list):
        for value in v:add_checks(value)
if checked:add_checks(d)
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
leaf={'type':'string',**({'minLength':1} if checked else {})}
for _ in range(16):leaf={'type':'array','items':leaf}
obj={'properties':{'rows':leaf},'required':['rows']}
def choice(value):return {'oneOf':[{'properties':{'kind':{'const':tag},'value':value},'required':['value']} for tag in ('a','b')]}
wide={'properties':{f'p{i}':{'type':'boolean'} for i in range(240)},'required':[f'p{i}' for i in range(240)]}
limits={'openapi':'3.1.0','components':{'schemas':{'Wide':choice(wide),'Checked':choice(obj)}}}
(p/'limits.json').write_text(json.dumps(limits,indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.inlinevalues --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.inlinevalues --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.inlinevaluelimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);checked=sys.argv[3]=='on'
copy='' if sys.argv[2]=='plain_class' else '''val value=org.example.inlinevalues.models.DirectDto.A.Value("ok")
    check(org.example.inlinevalues.models.DirectDto.A(value).copy(value=value.copy(id="next")).value.id=="next")'''
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
    val input=JsonObject(mapOf("kind" to JsonPrimitive("a"),"value" to JsonObject((0 until 240).associate { "p$it" to JsonPrimitive(true) })))
    val wide:org.example.inlinevaluelimits.models.WideDto=codec.decodeFromJsonElement(input)
    check((wide as org.example.inlinevaluelimits.models.WideDto.A).value.p0 && wide.value.p239 && codec.encodeToJsonElement(wide)==input)
    var leaf:kotlinx.serialization.json.JsonElement=JsonPrimitive("good")
    repeat(16) { leaf=JsonArray(listOf(leaf)) }
    val deep=JsonObject(mapOf("kind" to JsonPrimitive("a"),"value" to JsonObject(mapOf("rows" to leaf))))
    val typed:org.example.inlinevaluelimits.models.CheckedDto=codec.decodeFromJsonElement(deep)
    check(codec.encodeToJsonElement(typed)==deep)
    unionInlineValuesContracts('''+str(checked).lower()+''')
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runUnionInlineValuesContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runUnionInlineValuesContracts
    echo "FRESH UNION INLINE VALUES PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh inline union native projects: $workspace"
