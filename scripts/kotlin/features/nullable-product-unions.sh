#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/nullable-product-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullableProductUnionsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/nullable-product-unions.json').read_text())
def add_checks(v):
    if isinstance(v,dict):
        if v.get('type')=='string':v['minLength']=2
        for value in list(v.values()):add_checks(value)
    elif isinstance(v,list):
        for value in v:add_checks(value)
if checked:add_checks(d)
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
def choice(n):return {'oneOf':[{'properties':{'kind':{'const':'k'+str(i)}}} for i in range(n)]}
limits={'openapi':'3.1.0','components':{'schemas':{'Wide':{'nullable':True,'allOf':[choice(239)]},'Small':{'nullable':True,'allOf':[choice(2)]},'Hops':{'properties':{}}}}}
schemas=limits['components']['schemas'];ref='#/components/schemas/'
for i in range(45):schemas['Hops']['properties']['h'+str(i)]={'$ref':ref+('Small' if i==0 else 'Hops/properties/h'+str(i-1)),'nullable':False}
schemas['Near']={'properties':{'value':{'$ref':ref+'Hops/properties/h44','nullable':False}},'required':['value']}
(p/'limits.json').write_text(json.dumps(limits,indent=2)+'\n')
PYINPUT
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.nullableproducts --model-name-suffix Dto "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.nullableproducts --model-name-suffix Dto "${flags[@]}" --check
    "$workspace/openapi-kt" generate --input "$project/limits.json" --output "$project/build/generated-limits" --package-name org.example.nullableproductlimits --model-name-suffix Dto "${flags[@]}"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);checked=sys.argv[3]=='on'
copy='' if sys.argv[2]=='plain_class' else '''check(org.example.nullableproducts.models.ProductDto.A("old").copy(id="next").id=="next")'''
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
    check(codec.decodeFromString<org.example.nullableproductlimits.models.WideDto>("null") === org.example.nullableproductlimits.models.NullWideDto)
    val input=JsonObject(mapOf("kind" to JsonPrimitive("k238")))
    val wide:org.example.nullableproductlimits.models.WideDto=codec.decodeFromJsonElement(input)
    check(codec.encodeToJsonElement(wide)==input)
    val near=org.example.nullableproductlimits.models.NearDto(org.example.nullableproductlimits.models.SmallDto.K0)
    check(codec.decodeFromString<org.example.nullableproductlimits.models.NearDto>(codec.encodeToString(near))==near)
    nullableProductUnionsContracts('''+str(checked).lower()+''')
}
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runNullableProductUnionsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNullableProductUnionsContracts
    echo "FRESH NULLABLE PRODUCT UNIONS PROFILE $style $checked PASS: $project"
  done
done
echo "Fresh nullable product native projects: $workspace"
