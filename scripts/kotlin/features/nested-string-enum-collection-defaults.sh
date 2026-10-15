#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/nested-string-enum-collection-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedStringEnumCollectionDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
from copy import deepcopy
from pathlib import Path
import json
import sys

p = Path(sys.argv[1])
checked = sys.argv[3] == 'true'
d = json.loads(Path('tests/fixtures/http/nested-string-enum-collection-defaults.json').read_text())
if checked:
    fields = d['components']['schemas']['Holder']['properties']
    fields['required']['minItems'] = 1
    fields['required']['items']['minItems'] = 1
    fields['rows']['maxItems'] = 3
    fields['rows']['items']['maxItems'] = 4
    fields['listMap']['maxItems'] = 3
    fields['listMap']['items']['maxProperties'] = 3
    fields['mapList']['maxProperties'] = 3
    fields['mapList']['additionalProperties']['maxItems'] = 2
    fields['mapMap']['maxProperties'] = 2
    fields['mapMap']['additionalProperties']['maxProperties'] = 3
    fields['mixed']['items']['additionalProperties']['items']['maxProperties'] = 1
    required = d['components']['schemas']['RequiredTrees']['properties']
    required['rows']['minItems'] = 1
    required['rows']['items']['minItems'] = 1
    required['listMap']['minItems'] = 1
    required['listMap']['items']['minProperties'] = 1
    required['mapList']['minProperties'] = 1
    required['mapList']['additionalProperties']['minItems'] = 1
    required['mapMap']['minProperties'] = 1
    required['mapMap']['additionalProperties']['minProperties'] = 1
    patch = d['paths']['/patch']['patch']['requestBody']['content']['application/json']['schema']['properties']
    patch['required']['minItems'] = 1
    patch['required']['items']['minItems'] = 1
    patch['rows']['minItems'] = 1
    patch['rows']['items']['minItems'] = 1
    patch['listMap']['minItems'] = 1
    patch['listMap']['items']['maxProperties'] = 2
    patch['mapList']['minProperties'] = 1
    patch['mapList']['additionalProperties']['maxItems'] = 2
    patch['mapMap']['minProperties'] = 1
    patch['mapMap']['additionalProperties']['maxProperties'] = 2
(p/'input.json').write_text(json.dumps(d, ensure_ascii=False))

# Actual canonical lexical enums after dedup, merge and combined hoist/name-lock.
leaf = {'type': ['string', 'null'], 'enum': ['one', None, 'two']}
schema = {'properties': {
    'rows': {'type': 'array', 'items': {'type': 'array', 'items': deepcopy(leaf)}, 'default': [['one', None]]},
    'listMap': {'type': 'array', 'items': {'additionalProperties': deepcopy(leaf)}, 'default': [{'b': None, 'a': 'one'}]},
    'mapList': {'additionalProperties': {'type': 'array', 'items': deepcopy(leaf)}, 'default': {'b': [], 'a': ['two', None]}},
    'mapMap': {'additionalProperties': {'additionalProperties': deepcopy(leaf)}, 'default': {'b': {}, 'a': {'b': None, 'a': 'two'}}},
    'kotlin': {'type': 'string'}, 'kotlinx': {'type': 'string'},
    'companion': {'type': 'string'}, 'serializer': {'type': 'string'}}}
if checked:
    fields = schema['properties']
    fields['rows']['items']['maxItems'] = 2
    fields['listMap']['items']['maxProperties'] = 2
    fields['mapList']['additionalProperties']['maxItems'] = 2
    fields['mapMap']['additionalProperties']['maxProperties'] = 2
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': schema}}}}}} for name in ['alpha', 'beta']}
for mode in ['options', 'dedup', 'merge']:
    document = {'openapi': '3.1.0', 'paths': paths}
    if mode == 'merge':
        document['components'] = {'schemas': {'Alpha': schema, 'Zeta': schema}}
    (p/f'{mode}.json').write_text(json.dumps(document))

# Independent limits: each probe stresses exactly one budget.
def write_probe(mode, name, properties):
    (p/f'{mode}.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {name: {'properties': properties}}}}, separators=(',', ':')))

wide = {f'v{i}': {'type': 'array', 'items': {'type': 'array', 'items': {'enum': ['one', 'two']}}, 'default': [['one']]} for i in range(240)}
write_probe('slots', 'Wide', wide)
for mode, name, edges in [('source', 'DeepSource', 47), ('checked', 'DeepChecked', 16)]:
    chain = {'enum': ['one', 'two']}
    for _ in range(edges):
        chain = {'type': 'array', 'items': chain}
        if mode == 'checked': chain['minItems'] = 0
    chain['default'] = []
    write_probe(mode, name, {'value': chain})
compact = lambda value: json.dumps(value, separators=(',', ':'), ensure_ascii=False)
key_size = 32768 - len(compact([{'': 'one'}]).encode())
large = {'type': 'array', 'items': {'additionalProperties': {'enum': ['one', 'two']}}, 'default': [{'a'*key_size: 'one'}]}
assert len(compact(large['default']).encode()) == 32768
write_probe('bytes', 'Large', {'value': large})
vocabulary = {'type': 'array', 'items': {'type': 'array', 'items': {'type': ['string', 'null'], 'enum': [f'v{i}' for i in range(239)] + [None]}}, 'default': [['v238', None]]}
write_probe('vocabulary', 'Vocabulary', {'value': vocabulary})

copy = ''
if sys.argv[2] == 'data_class':
    copy = '''
 val value = HolderDto(required = listOf(listOf(HolderDto.RequiredInnerInner.TWO)))
 check(value.copy(kotlin = "copy", kotlinx = "copy", companion = "copy", serializer = "copy").rows == value.rows)
 check(value.copy(rows = listOf(listOf(HolderDto.RowsInnerInner.Unrecognized("future")))).rows?.single()?.single()?.serialName == "future")
 check(value.copy(listMap = listOf(mapOf("a" to HolderDto.ListMapInnerValue.Unrecognized("future")))).listMap?.single()?.get("a")?.serialName == "future")
 check(value.copy(mapList = mapOf("a" to listOf(HolderDto.MapListValueInner.Unrecognized("future")))).mapList?.get("a")?.single()?.serialName == "future")
 check(value.copy(mapMap = mapOf("a" to mapOf("b" to HolderDto.MapMapValueValue.Unrecognized("future")))).mapMap?.get("a")?.get("b")?.serialName == "future")
'''
    if checked:
        copy += '''
 check(runCatching { value.copy(required = emptyList()) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(required = listOf(emptyList())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(rows = List(4) { null }) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(rows = listOf(List(5) { null })) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(listMap = List(4) { null }) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(listMap = listOf(mapOf("a" to null, "b" to null, "c" to null, "d" to null))) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(mapList = mapOf("a" to null, "b" to null, "c" to null, "d" to null)) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(mapList = mapOf("a" to List(3) { null })) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(mapMap = mapOf("a" to emptyMap(), "b" to emptyMap(), "c" to emptyMap())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(mapMap = mapOf("a" to mapOf("a" to null, "b" to null, "c" to null, "d" to null))) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { PatchInputDto(required = null).copy(rows = Patch.Value(listOf(emptyList()))) }.exceptionOrNull() is IllegalArgumentException)
 val requiredTrees = RequiredTreesDto(rows = listOf(listOf(RequiredTreesDto.RowsInnerInner.TWO)), listMap = listOf(mapOf("a" to RequiredTreesDto.ListMapInnerValue.TWO)), mapList = mapOf("a" to listOf(RequiredTreesDto.MapListValueInner.TWO)), mapMap = mapOf("a" to mapOf("b" to RequiredTreesDto.MapMapValueValue.TWO)))
 check(runCatching { requiredTrees.copy(rows = listOf(emptyList())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { requiredTrees.copy(listMap = listOf(emptyMap())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { requiredTrees.copy(mapList = mapOf("a" to emptyList())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { requiredTrees.copy(mapMap = mapOf("a" to emptyMap())) }.exceptionOrNull() is IllegalArgumentException)
'''

owners = [
    ('Optimized', 'HoistedAlpha', 'HoistedBeta', 'org.example.nestedstringenumoptions.models.ResponseDto'),
    ('DedupAlpha', 'DedupAlpha', 'DedupBeta', 'org.example.nestedstringenumdedup.api.models.AlphaResponseDto'),
    ('MergedAlpha', 'MergedZeta', 'MergedResponse', 'org.example.nestedstringenummerge.models.AlphaDto')]
body = ''
for owner, first, second, descriptor in owners:
    body += f'''
 val {owner.lower()} = {owner}(kotlin = "shadow", kotlinx = "shadow", companion = "shadow", serializer = "shadow")
 val {owner.lower()}Rows: List<List<{owner}.RowsInnerInner?>>? = {owner.lower()}.rows
 val {owner.lower()}ListMap: List<Map<String, {owner}.ListMapInnerValue?>>? = {owner.lower()}.listMap
 val {owner.lower()}MapList: Map<String, List<{owner}.MapListValueInner?>>? = {owner.lower()}.mapList
 val {owner.lower()}MapMap: Map<String, Map<String, {owner}.MapMapValueValue?>>? = {owner.lower()}.mapMap
 check({owner.lower()}Rows == listOf(listOf({owner}.RowsInnerInner.ONE, null)))
 check({owner.lower()}ListMap == listOf(mapOf("a" to {owner}.ListMapInnerValue.ONE, "b" to null)))
 check({owner.lower()}MapList == mapOf("a" to listOf({owner}.MapListValueInner.TWO, null), "b" to emptyList()))
 check({owner.lower()}MapMap == mapOf("a" to mapOf("a" to {owner}.MapMapValueValue.TWO, "b" to null), "b" to emptyMap()))
 check({first}()::class == {second}()::class && {second}()::class == {owner}()::class)
 check({second}.serializer().descriptor.serialName == "{descriptor}")
 check({second}().rows == {owner}().rows && {second}().mapMap == {owner}().mapMap)
 check({owner}().rows !== {owner}().rows && {owner}().mapMap?.get("a") !== {owner}().mapMap?.get("a"))
'''
    for shape, member in [('RowsInnerInner', 'ONE'), ('ListMapInnerValue', 'ONE'), ('MapListValueInner', 'TWO'), ('MapMapValueValue', 'TWO')]:
        wire = member.lower()
        body += f'''
 check({owner}.{shape}.serializer().descriptor.serialName == "{descriptor}.{shape}")
 check({owner}.{shape}.fromSerialName("{wire}") === {owner}.{shape}.{member})
 check({owner}.{shape}.Serializer.fromSerialName("future") == {owner}.{shape}.Unrecognized("future"))
'''
    body += f'''
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {{
   val json = Json {{ encodeDefaults = defaults; explicitNulls = nulls }}
   val alias = json.decodeFromString({second}.serializer(), "{{}}")
   check(alias.rows == {owner}().rows && alias.listMap == {owner}().listMap && alias.mapList == {owner}().mapList && alias.mapMap == {owner}().mapMap)
   check(json.encodeToJsonElement({second}.serializer(), alias) == json.encodeToJsonElement({owner}.serializer(), {owner}()))
   check(json.decodeFromString({owner}.RowsInnerInner.serializer().nullable, "null") == null)
   check(json.decodeFromString({owner}.MapMapValueValue.serializer(), "\\\"future\\\"") == {owner}.MapMapValueValue.Unrecognized("future"))
'''
    if checked:
        for shape, invalid in [
            ('rows', '[["one","one","one"]]'),
            ('listMap', '[{"a":null,"b":null,"c":null}]'),
            ('mapList', '{"a":[null,null,null]}'),
            ('mapMap', '{"a":{"a":null,"b":null,"c":null}}')]:
            rule = 'maxItems' if shape in ('rows', 'mapList') else 'maxProperties'
            body += f'''
   val failure{shape} = runCatching {{ json.decodeFromString({second}.serializer(), """{{"{shape}":{invalid}}}""") }}.exceptionOrNull()
   check(generateSequence(failure{shape}) {{ it.cause }}.any {{ it is IllegalArgumentException && it.message.orEmpty().contains("{rule}") }})
'''
    body += ' }\n'
    if checked:
        body += f'''
 check(runCatching {{ {second}(rows = listOf(List(3) {{ null }})) }}.exceptionOrNull() is IllegalArgumentException)
 check(runCatching {{ {second}(listMap = listOf(mapOf("a" to null, "b" to null, "c" to null))) }}.exceptionOrNull() is IllegalArgumentException)
 check(runCatching {{ {second}(mapList = mapOf("a" to List(3) {{ null }})) }}.exceptionOrNull() is IllegalArgumentException)
 check(runCatching {{ {second}(mapMap = mapOf("a" to mapOf("a" to null, "b" to null, "c" to null))) }}.exceptionOrNull() is IllegalArgumentException)
'''

(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.builtins.nullable
import kotlinx.serialization.json.Json
import org.example.nestedstringenum.models.HolderDto
import org.example.nestedstringenum.models.RequiredTreesDto
import org.example.nestedstringenum.models.Patch
import org.example.nestedstringenum.api.models.PatchInputDto
import org.example.nestedstringenumoptions.models.ResponseDto as Optimized
import org.example.nestedstringenumoptions.api.models.AlphaResponseDto as HoistedAlpha
import org.example.nestedstringenumoptions.api.models.BetaResponseDto as HoistedBeta
import org.example.nestedstringenumdedup.api.models.AlphaResponseDto as DedupAlpha
import org.example.nestedstringenumdedup.api.models.BetaResponseDto as DedupBeta
import org.example.nestedstringenummerge.models.AlphaDto as MergedAlpha
import org.example.nestedstringenummerge.models.ZetaDto as MergedZeta
import org.example.nestedstringenummerge.api.models.AlphaResponseDto as MergedResponse
import org.example.nestedstringenumresources.models.WideDto
import org.example.nestedstringenumresources.models.DeepSourceDto
import org.example.nestedstringenumresources.models.DeepCheckedDto
import org.example.nestedstringenumresources.models.LargeDto
import org.example.nestedstringenumresources.models.VocabularyDto
fun main() = runBlocking {
''' + body + f'''
 check(WideDto.serializer().descriptor.elementsCount == 240)
 check(WideDto().v239 == listOf(listOf(WideDto.V239InnerInner.ONE)))
 check(DeepSourceDto().value?.isEmpty() == true && DeepCheckedDto().value?.isEmpty() == true)
 check(LargeDto().value?.single()?.keys?.single()?.length == {key_size})
 check(LargeDto().value?.single()?.values?.single() === LargeDto.ValueInnerValue.ONE)
 check(VocabularyDto.ValueInnerInner.entries.size == 239)
 check(VocabularyDto().value == listOf(listOf(VocabularyDto.ValueInnerInner.V238, null)))
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {{
   val json = Json {{ encodeDefaults = defaults; explicitNulls = nulls }}
   check(json.decodeFromString(WideDto.serializer(), "{{}}").v239 == WideDto().v239)
   check(json.decodeFromString(DeepSourceDto.serializer(), "{{}}").value?.isEmpty() == true)
   check(json.decodeFromString(DeepCheckedDto.serializer(), "{{}}").value?.isEmpty() == true)
   check(json.decodeFromString(LargeDto.serializer(), "{{}}").value == LargeDto().value)
   check(json.decodeFromString(VocabularyDto.serializer(), "{{}}").value == VocabularyDto().value)
 }}
''' + copy + ' nestedStringEnumCollectionDefaultsContracts(' + sys.argv[3] + ')\n}\n')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
// Keep the full source48 lexical enum names in an archive: nested binary
// names can exceed the host filesystem's per-component NAME_MAX.
tasks.named<org.jetbrains.kotlin.gradle.tasks.KotlinCompile>("compileKotlin") {
 destinationDirectory.set(layout.buildDirectory.dir("classes/kotlin/main.jar"))
 incremental = false
 doFirst {
  val output = destinationDirectory.get().asFile
  if (output.isDirectory) {
   check(output.list()?.isEmpty() == true)
   check(output.delete())
  }
 }
}
tasks.register<JavaExec>("runNestedStringEnumCollectionDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath + files(layout.buildDirectory.file("classes/kotlin/main.jar"))
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.nestedstringenum --api-client-name NestedStringEnum --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.nestedstringenumoptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    for mode in dedup merge slots source checked bytes vocabulary; do
      optimization=()
      package="org.example.nestedstringenumresources"
      if [[ "$mode" == dedup ]]; then optimization=(--deduplicate-models); package="org.example.nestedstringenumdedup"; fi
      if [[ "$mode" == merge ]]; then optimization=(--merge-identical-models); package="org.example.nestedstringenummerge"; fi
      if [[ "$checked" == true || "$mode" == checked ]]; then optimization+=(--validation); fi
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "$package" --model-name-suffix Dto --model-style "$style")
      if (( ${#optimization[@]} > 0 )); then mode_flags+=("${optimization[@]}"); fi
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED NESTED STRING ENUM COLLECTION DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNestedStringEnumCollectionDefaultsContracts
    echo "FRESH NESTED STRING ENUM COLLECTION DEFAULTS $style checked=$checked PASS: $project"
  done
done
