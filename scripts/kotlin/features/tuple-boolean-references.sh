#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-boolean-references-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleBooleanReferencesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    flags=(--model-style "$style")
    if [[ "$checked" == on ]]; then flags+=(--validation); fi
    python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]); checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/tuple-boolean-references.json').read_text())
if checked:
    d['components']['schemas']['Holder']['properties']['open']['maxItems']=2
    d['components']['schemas']['Holder']['properties']['nullable']['maxItems']=1
# Independent fields sharing actual false-root codecs and empty-default helpers at the JVM240 cap.
d['components']['schemas']['Wide']={'properties':{f'v{i:03d}':{'type':'array','prefixItems':[{'$ref':'#/components/schemas/Denied'}],'default':[]} for i in range(240)}}
# A separate directional consumer verifies the existing read/write projections
# carry actual rejecting tuple codecs, independent of the shared fixture API.
d['components']['schemas']['Direction']={'properties':{
    'readRows':{'type':'array','prefixItems':[{'$ref':'#/components/schemas/Denied'}],'readOnly':True},
    'writeRows':{'type':'array','prefixItems':[{'$ref':'#/components/schemas/Denied'}],'writeOnly':True}}}
d['paths']['/direction']={'post':{'operationId':'direction',
    'requestBody':{'required':True,'content':{'application/json':{'schema':{'$ref':'#/components/schemas/Direction'}}}},
    'responses':{'200':{'description':'synthetic result','content':{'application/json':{'schema':{'$ref':'#/components/schemas/Direction'}}}}}}}
(p/'input.json').write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
consumer=p/'src/main/kotlin/org/example/contracts/TupleBooleanReferencesContracts.kt'
(p/'src/main/kotlin/org/example/contracts/OptimizedTupleBooleanReferencesContracts.kt').write_text(consumer.read_text().replace('org.example.booleantuplerefs','org.example.booleantuplerefsoptimized').replace('BooleanReferenceStreamEngine','OptimizedBooleanReferenceStreamEngine').replace('booleanReference','optimizedBooleanReference').replace('tupleBooleanReferences','optimizedTupleBooleanReferences'))
directional = r'''package org.example.contracts
import io.ktor.client.HttpClient
import io.ktor.client.engine.mock.MockEngine
import io.ktor.client.engine.mock.respond
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.http.headersOf
import io.ktor.http.content.OutgoingContent
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.jsonObject
import org.example.booleantuplerefs.api.DefaultApi
import org.example.booleantuplerefs.api.models.DirectionRequestDto
private fun directionBody(content: OutgoingContent): String = when(content) {
    is OutgoingContent.ContentWrapper -> directionBody(content.delegate())
    is OutgoingContent.ByteArrayContent -> content.bytes().decodeToString()
    else -> error("Unexpected synthetic directional tuple body")
}
suspend fun booleanTupleDirections() {
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val codec = Json { encodeDefaults=defaults; explicitNulls=nulls }
        var bad = false
        val client = HttpClient(MockEngine { request ->
            val body=codec.parseToJsonElement(directionBody(request.body)).jsonObject
            check("readRows" !in body && body["writeRows"] == JsonArray(emptyList()))
            respond(if(bad) """{"readRows":[null]}""" else """{"readRows":[]}""", headers=headersOf("Content-Type","application/json"))
        }) { install(ContentNegotiation) { json(codec) } }
        try {
            val api=DefaultApi(client,"https://example.test/",codec)
            check(api.direction(DirectionRequestDto(writeRows=emptyList())).readRows!!.isEmpty())
            bad=true
            check(runCatching { api.direction(DirectionRequestDto(writeRows=emptyList())) }.isFailure)
        } finally {client.close()}
    }
}
'''
(p/'src/main/kotlin/org/example/contracts/BooleanTupleDirectional.kt').write_text(directional)
(p/'src/main/kotlin/org/example/contracts/OptimizedBooleanTupleDirectional.kt').write_text(directional.replace('org.example.booleantuplerefs','org.example.booleantuplerefsoptimized').replace('directionBody','optimizedDirectionBody').replace('booleanTupleDirections','optimizedBooleanTupleDirections'))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
    tupleBooleanReferencesContracts('''+str(checked).lower()+''')
    optimizedTupleBooleanReferencesContracts('''+str(checked).lower()+''')
    booleanTupleDirections()
    optimizedBooleanTupleDirections()
}
''')
build=p/'build.gradle.kts'
build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runTupleBooleanReferencesContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
    for mode in baseline optimized; do
      output="$project/build/generated"
      package=org.example.booleantuplerefs
      options=()
      if [[ "$mode" == optimized ]]; then
        output="$project/build/generated-limits"
        package=org.example.booleantuplerefsoptimized
        options=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate --input "$project/input.json" --output "$output" --package-name "$package" --model-name-suffix Dto "${flags[@]}" "${options[@]}" --check
    done
    python3 - "$project/build/generated-limits/.openapi-kt-names.json" <<'PYLOCK'
import json,sys
assert json.load(open(sys.argv[1]))['entries'], 'optimized tuple profile requires actual locked hoist allocations'
PYLOCK
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleBooleanReferencesContracts
    echo "FRESH BOOLEAN TUPLE REFERENCES PROFILE $style $checked PASS: $project"
  done
done
