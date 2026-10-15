#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/union-common-dictionary-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/UnionCommonDictionariesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/union-common-dictionaries.json').read_text())
if checked:
    branches=d['components']['schemas']['Checked']['oneOf']
    branches[0]['properties']['value'].update(minProperties=1)
    branches[0]['properties']['value']['additionalProperties']['minLength']=2
    branches[1]['properties']['value'].update(maxProperties=2)
    branches[1]['properties']['value']['additionalProperties']['minLength']=3
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
def choice(props,required):
    return {'oneOf':[{'properties':{'kind':{'const':tag},**props},'required':required} for tag in ('a','b')]}
# A pinned tag occupies one serialized slot; common Map fields remain one slot.
props={f'p{i}':{'additionalProperties':{'type':'string'}} for i in range(239)}
wide=choice(props,list(props))
leaf={'type':'string'}
if checked: leaf['minLength']=1
for _ in range(16):leaf={'additionalProperties':leaf}
limits={'Wide':wide,'Checked':choice({'value':leaf},['value'])}
(p/'limits.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':limits}},indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.commondictionaries --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.commondictionaries --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.commondictionarylimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);checked=sys.argv[3]=='on'
copy='' if sys.argv[2]=='plain_class' else 'check(org.example.commondictionaries.models.DirectDto.A(mapOf("key" to "one")).copy(value=mapOf("key" to "two")).value.getValue("key")=="two")'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.encodeToJsonElement
fun main()=runBlocking {
    '''+copy+'''
    val codec=Json
    val input=JsonObject(mapOf("kind" to JsonPrimitive("a"))+(0 until 239).associate { "p$it" to JsonObject(mapOf("key" to JsonPrimitive("value"))) })
    val wide:org.example.commondictionarylimits.models.WideDto=codec.decodeFromJsonElement(input)
    check(wide.p0.getValue("key")=="value" && wide.p238.getValue("key")=="value")
    check(codec.encodeToJsonElement(wide)==input)
    var leaf:kotlinx.serialization.json.JsonElement=JsonPrimitive("good")
    repeat(16) { leaf=JsonObject(mapOf("key" to leaf)) }
    val deep=JsonObject(mapOf("kind" to JsonPrimitive("a"),"value" to leaf))
    val typed:org.example.commondictionarylimits.models.CheckedDto=codec.decodeFromJsonElement(deep)
    check(codec.encodeToJsonElement(typed)==deep)
    unionCommonDictionariesContracts('''+str(checked).lower()+''')
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runUnionCommonDictionariesContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runUnionCommonDictionariesContracts
    echo "FRESH UNION COMMON DICTIONARIES PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh common dictionary native projects: $workspace"
