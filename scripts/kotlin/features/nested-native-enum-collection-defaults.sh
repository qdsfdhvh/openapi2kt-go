#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/nested-native-enum-collection-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NestedNativeEnumCollectionDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
from copy import deepcopy
from pathlib import Path
import json
import sys

p = Path(sys.argv[1])
checked = sys.argv[3] == 'true'
d = json.loads(Path('tests/fixtures/http/nested-native-enum-collection-defaults.json').read_text())

def visit(schema):
    yield schema
    for key in ('items', 'additionalProperties'):
        if isinstance(schema.get(key), dict):
            yield from visit(schema[key])
    for field in schema.get('properties', {}).values():
        yield from visit(field)

# Only the checked variant uses bounds/counts and the validation-only raw-null
# vocabulary grammar. All five native leaf types keep their real codecs.
if checked:
    for schema in visit(d['components']['schemas']['Holder']):
        if 'enum' in schema and schema.get('format') in ('int32', 'int64') and schema['enum'] == [1, 2, 3]:
            schema.update(minimum=2, maximum=3, multipleOf=1)
        if 'enum' in schema and schema.get('format') in ('float', 'double') and schema['enum'] == [0.5, 1, 1.5]:
            schema.update(minimum=0.5, maximum=1)
    fields = d['components']['schemas']['Holder']['properties']
    for native in ('int', 'long', 'float', 'double', 'bool'):
        rows, lm, ml, mm, mixed = (fields[native + shape] for shape in ('Rows', 'ListMap', 'MapList', 'MapMap', 'Mixed'))
        rows['maxItems'] = 3
        rows['items']['maxItems'] = 2
        lm['maxItems'] = 3
        lm['items']['maxProperties'] = 2
        ml['maxProperties'] = 3
        ml['additionalProperties']['maxItems'] = 2
        mm['maxProperties'] = 3
        mm['additionalProperties']['maxProperties'] = 2
        mixed['items']['additionalProperties']['items']['maxProperties'] = 1
    for schema in visit(d['components']['schemas']['Holder']):
        nullable = schema.get('nullable') is True or (isinstance(schema.get('type'), list) and 'null' in schema['type'])
        if 'enum' in schema and nullable:
            schema['enum'].insert(1, None)
    for owner in (d['components']['schemas']['RequiredTrees'], d['paths']['/patch']['patch']['requestBody']['content']['application/json']['schema']):
        for schema in visit(owner):
            if schema.get('type') == 'array': schema.update(minItems=1, maxItems=2)
            if 'additionalProperties' in schema: schema.update(minProperties=1, maxProperties=2)
            if schema.get('enum') == [1, 2, 3]: schema.update(minimum=2, maximum=3, multipleOf=1)
            if schema.get('enum') == [0.5, 1, 1.5]: schema.update(minimum=0.5, maximum=1)
    fields['required']['minItems'] = 1
    fields['required']['items']['minItems'] = 1
    operation = d['paths']['/native']['post']
    for schema in (operation['requestBody']['content']['application/json']['schema'], operation['responses']['200']['content']['application/json']['schema']):
        schema['minItems'] = 1
        for node in visit(schema):
            if 'enum' in node: node['minimum'] = 2
(p/'input.json').write_text(json.dumps(d, ensure_ascii=False))

# Real canonical native collection owners consumed through each actual alias.
fields = d['components']['schemas']['Holder']['properties']
schema = {'properties': {name: deepcopy(value) for name, value in fields.items() if any(name == native + shape for native in ('int', 'long', 'float', 'double', 'bool') for shape in ('Rows', 'ListMap', 'MapList', 'MapMap', 'Mixed')) or name in ('kotlin', 'kotlinx', 'companion', 'serializer')}}
paths = {f'/{name}': {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': schema}}}}}} for name in ('alpha', 'beta')}
for mode in ('options', 'dedup', 'merge'):
    document = {'openapi': '3.1.0', 'paths': paths}
    if mode == 'merge': document['components'] = {'schemas': {'Alpha': schema, 'Zeta': schema}}
    (p/f'{mode}.json').write_text(json.dumps(document))

# Independent compiler inputs. Never trade depth for width or sum their caps.
compact = lambda value: json.dumps(value, separators=(',', ':'), ensure_ascii=False)
def write_probe(mode, name, properties):
    (p/f'{mode}.json').write_text(compact({'openapi': '3.1.0', 'components': {'schemas': {name: {'properties': properties}}}}))

def native_leaf(): return {'type': 'integer', 'format': 'int32', 'enum': [1, 2]}
wide = {f'v{i}': {'type': 'array', 'items': {'type': 'array', 'items': native_leaf()}, 'default': [[2]]} for i in range(240)}
assert len(wide) == 240
write_probe('slots', 'Wide', wide)
for mode, name, edges in (('source', 'DeepSource', 47), ('checked', 'DeepChecked', 16)):
    chain = native_leaf()
    for _ in range(edges):
        chain = {'type': 'array', 'items': chain}
        if mode == 'checked': chain['minItems'] = 0
    chain['default'] = []
    write_probe(mode, name, {'value': chain})
key_size = 32768 - len(compact([{'': 2}]).encode())
large = {'type': 'array', 'items': {'additionalProperties': native_leaf()}, 'default': [{'a'*key_size: 2}]}
assert len(compact(large['default']).encode()) == 32768
write_probe('bytes', 'Large', {'value': large})
raw = list(range(239)) + [None] if checked else list(range(240))
assert len(raw) == 240
last = 238 if checked else 239
vocabulary = {'type': 'array', 'items': {'type': 'array', 'items': {'type': ['integer', 'null'], 'format': 'int32', 'enum': raw}}, 'default': [[last, None]]}
write_probe('vocabulary', 'Vocabulary', {'value': vocabulary})

natives = [('int', 'Int', '2', '9'), ('long', 'Long', '2L', '9L'), ('float', 'Float', '1.0f', '9.0f'), ('double', 'Double', '1.0', '9.0'), ('bool', 'Boolean', 'true', 'false')]
shapes = [
    ('Rows', 'List<List<{t}?>?>?', 'listOf(listOf({v}, null), null, emptyList())', 'listOf(listOf({v}))'),
    ('ListMap', 'List<Map<String, {t}?>?>?', 'listOf(mapOf("\\$ref" to {v}, "a/~" to null), null, emptyMap())', 'listOf(mapOf("a" to {v}))'),
    ('MapList', 'Map<String, List<{t}?>?>?', 'mapOf("a" to listOf({v}, null), "nil" to null, "empty" to emptyList())', 'mapOf("a" to listOf({v}))'),
    ('MapMap', 'Map<String, Map<String, {t}?>?>?', 'mapOf("a" to mapOf("\\$ref" to {v}, "a/~" to null), "nil" to null, "empty" to emptyMap())', 'mapOf("a" to mapOf("b" to {v}))'),
    ('Mixed', 'List<Map<String, List<Map<String, {t}>>>>?', 'listOf(mapOf("\\$ref" to listOf(mapOf("a/~" to {v}))))', 'listOf(mapOf("a" to listOf(mapOf("b" to {v}))))')]
body = ''
for owner, first, second, descriptor in [
    ('Optimized', 'HoistedAlpha', 'HoistedBeta', 'org.example.nestednativeenumoptions.models.ResponseDto'),
    ('DedupAlpha', 'DedupAlpha', 'DedupBeta', 'org.example.nestednativeenumdedup.api.models.AlphaResponseDto'),
    ('MergedAlpha', 'MergedZeta', 'MergedResponse', 'org.example.nestednativeenummerge.models.AlphaDto')]:
    local = owner.lower()
    body += f'''
 val {local} = {owner}(kotlin = "shadow", kotlinx = "shadow", companion = "shadow", serializer = "shadow")
 check({first}()::class == {second}()::class && {second}()::class == {owner}()::class)
 check({second}.serializer().descriptor.serialName == "{descriptor}")
'''
    for native, scalar, good, unknown in natives:
        for shape, generic, expected, supplied in shapes:
            field = native + shape
            body += f' val {local}{field}: {generic.format(t=scalar)} = {local}.{field}\n'
            body += f' check({local}{field} == {expected.format(v=good)})\n'
            body += f' check({second}().{field} == {local}{field})\n'
            if checked:
                body += f' check(runCatching {{ {second}({field} = {supplied.format(v=unknown)}) }}.exceptionOrNull() is IllegalArgumentException)\n'
    body += f'''
 check({owner}().intRows !== {owner}().intRows && {owner}().doubleMapMap?.get("a") !== {owner}().doubleMapMap?.get("a"))
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {{
   val json = Json {{ encodeDefaults = defaults; explicitNulls = nulls }}
   val alias = json.decodeFromString({second}.serializer(), "{{}}")
   check(json.encodeToJsonElement({second}.serializer(), alias) == json.encodeToJsonElement({owner}.serializer(), {owner}()))
'''
    if checked:
        for field, invalid, rule in [('intRows', '[[9]]', 'enum'), ('longListMap', '[{"a":1}]', 'minimum'), ('floatMapList', '{"a":[1.5]}', 'maximum'), ('doubleMapMap', '{"a":{"a":null,"b":null,"c":null}}', 'maxProperties'), ('boolMixed', '[{"a":[{"b":false}]}]', 'enum')]:
            body += f'''
   val failure{field} = runCatching {{ json.decodeFromString({second}.serializer(), """{{"{field}":{invalid}}}""") }}.exceptionOrNull()
   check(generateSequence(failure{field}) {{ it.cause }}.any {{ it is IllegalArgumentException && it.message.orEmpty().contains("{rule}") }})
'''
    body += ' }\n'

body += '''
 val value = HolderDto(required = listOf(listOf(2)))
'''
# Each constructor/Plain path and each Data copy really consumes all five leaves
# across all four forms and the longer path, including unknown runtime values.
for native, scalar, good, unknown in natives:
    for shape, generic, expected, supplied in shapes:
        field = native + shape
        expression = supplied.format(v=unknown)
        if checked:
            body += f' check(runCatching {{ HolderDto(required = value.required, {field} = {expression}) }}.exceptionOrNull() is IllegalArgumentException)\n'
        else:
            body += f' check(HolderDto(required = value.required, {field} = {expression}).{field} == {expression})\n'
        if sys.argv[2] == 'data_class':
            if checked:
                body += f' check(runCatching {{ value.copy({field} = {expression}) }}.exceptionOrNull() is IllegalArgumentException)\n'
            else:
                body += f' check(value.copy({field} = {expression}).{field} == {expression})\n'
if sys.argv[2] == 'data_class':
    body += '''
 check(value.copy(kotlin = "copy", kotlinx = "copy", companion = "copy", serializer = "copy").longMapMap == value.longMapMap)
'''
    if checked:
        body += '''
 check(runCatching { value.copy(required = listOf(emptyList())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(intRows = List(4) { null }) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(longListMap = listOf(mapOf("a" to 1L))) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(floatMapList = mapOf("a" to listOf(1.5f))) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(doubleMapMap = mapOf("a" to mapOf("a" to null, "b" to null, "c" to null))) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { PatchInputDto(required = null).copy(rows = Patch.Value(listOf(emptyList()))) }.exceptionOrNull() is IllegalArgumentException)
 val trees = RequiredTreesDto(rows = listOf(listOf(2)), listMap = listOf(mapOf("a" to 2L)), mapList = mapOf("a" to listOf(1.0f)), mapMap = mapOf("a" to mapOf("b" to 1.0)), mixed = listOf(mapOf("a" to listOf(mapOf("b" to true)))))
 check(runCatching { trees.copy(rows = listOf(emptyList())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { trees.copy(listMap = listOf(emptyMap())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { trees.copy(mapList = mapOf("a" to emptyList())) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { trees.copy(mapMap = mapOf("a" to emptyMap())) }.exceptionOrNull() is IllegalArgumentException)
'''

(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import org.example.nestednativeenum.models.HolderDto
import org.example.nestednativeenum.models.RequiredTreesDto
import org.example.nestednativeenum.models.Patch
import org.example.nestednativeenum.api.models.PatchInputDto
import org.example.nestednativeenumoptions.models.ResponseDto as Optimized
import org.example.nestednativeenumoptions.api.models.AlphaResponseDto as HoistedAlpha
import org.example.nestednativeenumoptions.api.models.BetaResponseDto as HoistedBeta
import org.example.nestednativeenumdedup.api.models.AlphaResponseDto as DedupAlpha
import org.example.nestednativeenumdedup.api.models.BetaResponseDto as DedupBeta
import org.example.nestednativeenummerge.models.AlphaDto as MergedAlpha
import org.example.nestednativeenummerge.models.ZetaDto as MergedZeta
import org.example.nestednativeenummerge.api.models.AlphaResponseDto as MergedResponse
import org.example.nestednativeenumresources.models.WideDto
import org.example.nestednativeenumresources.models.DeepSourceDto
import org.example.nestednativeenumresources.models.DeepCheckedDto
import org.example.nestednativeenumresources.models.LargeDto
import org.example.nestednativeenumresources.models.VocabularyDto
fun main() = runBlocking {
''' + body + f'''
 check(WideDto.serializer().descriptor.elementsCount == 240)
 check(WideDto().v239 == listOf(listOf(2)))
 check(DeepSourceDto().value?.isEmpty() == true && DeepCheckedDto().value?.isEmpty() == true)
 check(LargeDto().value?.single()?.keys?.single()?.length == {key_size})
 check(LargeDto().value?.single()?.values?.single() == 2)
 val vocabulary: List<List<Int?>>? = VocabularyDto().value
 check(vocabulary == listOf(listOf({last}, null)))
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {{
   val json = Json {{ encodeDefaults = defaults; explicitNulls = nulls }}
   check(json.decodeFromString(WideDto.serializer(), "{{}}").v239 == WideDto().v239)
   check(json.decodeFromString(DeepSourceDto.serializer(), "{{}}").value?.isEmpty() == true)
   check(json.decodeFromString(DeepCheckedDto.serializer(), "{{}}").value?.isEmpty() == true)
   check(json.decodeFromString(LargeDto.serializer(), "{{}}").value == LargeDto().value)
   check(json.decodeFromString(VocabularyDto.serializer(), "{{}}").value == vocabulary)
 }}
 nestedNativeEnumCollectionDefaultsContracts({sys.argv[3]})
}}
''')
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
// Native terminals create no deep lexical binary names; use normal class output
// and the existing Kotlin DSL -Xuse-type-table option, with no depth shrink.
tasks.register<JavaExec>("runNestedNativeEnumCollectionDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.nestednativeenum --api-client-name NestedNativeEnum --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.nestednativeenumoptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    for mode in dedup merge slots source checked bytes vocabulary; do
      optimization=()
      package="org.example.nestednativeenumresources"
      if [[ "$mode" == dedup ]]; then optimization=(--deduplicate-models); package="org.example.nestednativeenumdedup"; fi
      if [[ "$mode" == merge ]]; then optimization=(--merge-identical-models); package="org.example.nestednativeenummerge"; fi
      # Native membership itself consumes checked depth. The independent full
      # source48 input must stay unchecked, including in checked consumer runs.
      if [[ "$mode" == checked || ( "$checked" == true && "$mode" != source ) ]]; then optimization+=(--validation); fi
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "$package" --model-name-suffix Dto --model-style "$style")
      # Bash 3.2 + nounset cannot expand an empty array safely.
      if (( ${#optimization[@]} > 0 )); then mode_flags+=("${optimization[@]}"); fi
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED NESTED NATIVE ENUM COLLECTION DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNestedNativeEnumCollectionDefaultsContracts
    echo "FRESH NESTED NATIVE ENUM COLLECTION DEFAULTS $style checked=$checked PASS: $project"
  done
done
