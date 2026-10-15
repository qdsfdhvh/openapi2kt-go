#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/direct-nullable-component-counts-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp tests/fixtures/http/direct-nullable-component-counts.json "$project/counted.json"
    if [[ "$checked" == true ]]; then
      cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DirectNullableComponentCountsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    fi
    python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import copy, json, sys
p = Path(sys.argv[1]); checked = sys.argv[3] == 'true'; ref = '#/components/schemas/'
d = json.loads((p/'counted.json').read_text())
def remove_checks(v):
    if isinstance(v, dict):
        for key in ['minProperties', 'maxProperties', 'minLength', 'maxLength']:
            v.pop(key, None)
        for child in v.values(): remove_checks(child)
    elif isinstance(v, list):
        for child in v: remove_checks(child)
if not checked: remove_checks(d)
(p/'input.json').write_text(json.dumps(d))
counted = {'type': ['object', 'null'], 'properties': {'value': {'type': 'string'}}}
if checked: counted.update(minProperties=0, maxProperties=1)
model = {'properties': {'payload': {'$ref': ref+'Record'}, 'owned': copy.deepcopy(counted)}}
paths = {f'/{n}': {'get': {'operationId': n, 'responses': {'200': {'content': {'application/json': {'schema': model}}}}}} for n in ['alpha', 'beta']}
(p/'options.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'Record': counted, 'Twin': counted}}, 'paths': paths}))
# Independent source48, checked16 and shallow240 probes preserve existing caps.
s = {'Record': counted, 'Wide': {'type': ['object', 'null'], 'properties': {f'v{i}': {'type': 'string'} for i in range(240)}}}
s['Longs'] = {'nullable': True, 'required': [f'v{i}' for i in range(120)], 'properties': {f'v{i}': {'type': 'integer', 'format': 'int64'} for i in range(120)}}
if checked:
    s['Wide'].update(minProperties=0, maxProperties=240)
    s['Longs'].update(minProperties=0, maxProperties=120)
for i in range(46): s[f'A{i:02d}'] = {'$ref': ref+('Record' if i == 0 else f'A{i-1:02d}')}
s['Source48'] = {'properties': {'value': {'$ref': ref+'A45', 'nullable': False}}}
leaf = {'type': 'string', **({'minLength': 1} if checked else {})}
for _ in range(16): leaf = {'type': 'array', 'items': leaf}
s['Deep'] = {'nullable': True, 'properties': {'value': leaf}}
if checked: s['Deep'].update(minProperties=0, maxProperties=1)
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': s}}))
copy_calls = ''
if sys.argv[2] == 'data_class':
    copy_calls = '''
 val empty = org.example.directcounts.models.RecordDto().copy()
 check(empty.value == null)
 val valid = org.example.directcounts.models.CheckedDto("ok")
 check(valid.copy(value="abc").value == "abc")
'''
    if checked:
        copy_calls += '''
 check(runCatching { valid.copy(value="x") }.exceptionOrNull() is IllegalArgumentException)
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
  val json = Json { encodeDefaults=defaults; explicitNulls=nulls }
  if (!(defaults && nulls)) check(runCatching { json.encodeToString(org.example.directcounts.models.RecordDto.serializer(), empty) }.exceptionOrNull() is kotlinx.serialization.SerializationException)
 }
'''
else:
    copy_calls = '''
 check(org.example.directcounts.models.RecordDto().javaClass.methods.none { it.name == "copy" || it.name.startsWith("component") })
'''
checked_calls = ''' directNullableComponentCountsContracts()
''' if checked else '''
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
  val codec = Json { encodeDefaults=defaults; explicitNulls=nulls; ignoreUnknownKeys=true }
  val physical = org.example.directcounts.models.RecordDto.serializer()
  check(!physical.descriptor.isNullable)
  check(codec.decodeFromString(physical, "{}") == org.example.directcounts.models.RecordDto())
  check(codec.decodeFromString(physical.nullable, "null") == null)
  check(codec.encodeToJsonElement(physical.nullable, null) == JsonNull)
  val record = org.example.directcounts.models.RecordDto("ok")
  val holder = org.example.directcounts.models.HolderDto(narrow=record, records=listOf(null,record), index=mapOf("x" to null))
  check(codec.decodeFromString(org.example.directcounts.models.HolderDto.serializer(), codec.encodeToString(org.example.directcounts.models.HolderDto.serializer(), holder)) == holder)
  val patch = org.example.directcounts.api.models.PatchInputDto(null, org.example.directcounts.models.Patch.Null)
  check((codec.encodeToJsonElement(org.example.directcounts.api.models.PatchInputDto.serializer(), patch) as JsonObject)["note"] == JsonNull)
  check(codec.decodeFromString(org.example.directcounts.models.DefaultedDto.serializer(), "{}").value == "ready")
  check(org.example.directcounts.models.CheckedDto("x").value == "x")
 }
 println("Direct nullable count validation-off no-count controls passed")
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
fun main() = runBlocking {
 val record = org.example.directcountoptions.models.RecordDto("ok")
 val nested = org.example.directcountoptions.models.ResponseDto.Owned("ok")
 val optimized = org.example.directcountoptions.models.ResponseDto(record, nested)
 check(org.example.directcountoptions.api.models.AlphaResponseDto(record, nested) == optimized)
 check(org.example.directcountoptions.api.models.BetaResponseDto(record, nested) == optimized)
 check(org.example.directcountoptions.models.TwinDto("ok") == record)
 check(org.example.directcountoptions.models.TwinDto.serializer().descriptor.serialName == "org.example.directcountoptions.models.RecordDto")
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
  val codec = Json { encodeDefaults=defaults; explicitNulls=nulls }
  check(codec.decodeFromString(org.example.directcountoptions.models.ResponseDto.serializer(), codec.encodeToString(org.example.directcountoptions.models.ResponseDto.serializer(), optimized)) == optimized)
  check(codec.decodeFromString(org.example.directcountoptions.models.RecordDto.serializer().nullable, "null") == null)
 }
 check(org.example.directcountresources.models.WideDto().v239 == null)
 check(org.example.directcountresources.models.LongsDto.serializer().descriptor.elementsCount == 120)
 val source = org.example.directcountresources.models.RecordDto("ok")
 check(org.example.directcountresources.models.Source48Dto(source).value == source)
 check(org.example.directcountresources.models.DeepDto().value == null)
'''+copy_calls+checked_calls+'}\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runDirectNullableComponentCountsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.directcounts --api-client-name DirectCounts --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.directcountoptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    resources=(--input "$project/resources.json" --output "$project/build/generated-limits/resources" --package-name org.example.directcountresources --model-name-suffix Dto --model-style "$style")
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); resources+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    "$workspace/openapi-kt" generate "${resources[@]}"
    "$workspace/openapi-kt" generate "${resources[@]}" --check
    if [[ "$checked" == false ]]; then
      "$workspace/openapi-kt" validate --input "$project/counted.json" --diagnostic-format json > "$project/validation-off.json" && exit 1
      python3 - "$project/validation-off.json" <<'PY'
import json, sys
r=json.load(open(sys.argv[1]));assert r['status']=='error' and r['diagnostics'][0]['code']=='E_SCHEMA_UNSUPPORTED'
assert r['diagnostics'][0]['pointer']=='/components/schemas/Record/minProperties'
PY
    fi
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED DIRECT NULLABLE COMPONENT COUNTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runDirectNullableComponentCountsContracts
    echo "FRESH DIRECT NULLABLE COMPONENT COUNTS $style checked=$checked PASS: $project"
  done
done
printf 'Fresh direct nullable component count projects: %s\n' "$workspace"
