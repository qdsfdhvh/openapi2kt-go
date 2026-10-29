#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
prepare_only=false
if [[ "${1:-}" == --prepare-only && $# -eq 1 ]]; then prepare_only=true
elif [[ $# -ne 0 ]]; then printf 'Usage: bash scripts/kotlin/features/string-enum-collection-bounds.sh [--prepare-only]\n' >&2; exit 2; fi
mkdir -p build
workspace="$(mktemp -d "$root/build/string-enum-collection-bounds-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/StringEnumCollectionBoundsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/string-enum-collection-bounds.json "$project/bounded.json"
    python3 - "$project" "$style" "$checked" <<'PYMAIN'
from pathlib import Path
import copy
import json
import sys
p = Path(sys.argv[1])
checked = sys.argv[3] == 'true'
d = json.loads((p/'bounded.json').read_text())
# Validation-off is an accepted unbounded control with the same lexical shape.
# Bounds in the original source must still reject without validation.
def remove_checks(v):
    if isinstance(v, dict):
        for key in ['minLength', 'maxLength', 'pattern', 'minItems', 'maxItems', 'minProperties', 'maxProperties', 'uniqueItems']:
            v.pop(key, None)
        for child in v.values():
            remove_checks(child)
    elif isinstance(v, list):
        for child in v:
            remove_checks(child)
if not checked:
    remove_checks(d)
(p/'input.json').write_text(json.dumps(d))
schema = {'properties': {
    'state': {'type': 'string', 'format': 'date', 'enum': ['one', 'two', 'x'], 'default': 'one'},
    'owned': {'$ref': '#/components/schemas/SeedAlias', 'default': {'state': 'two'}},
    'ownedRows': {'type': 'array', 'items': {'$ref': '#/components/schemas/SeedAlias'}, 'default': [{'state': 'two'}]},
    'codes': {'type': 'array', 'items': {'enum': ['one', 'two', 'x']}, 'default': ['one']},
    'labels': {'additionalProperties': {'enum': ['yes', 'no']}, 'default': {'b': 'no', 'a': 'yes'}}}}
if checked:
    schema['properties']['state'].update(minLength=2, maxLength=6, pattern='^[a-z]+$')
    schema['properties']['codes']['items'].update(minLength=2, maxLength=6, pattern='^[a-z]+$')
    schema['properties']['codes']['maxItems'] = 2
    schema['properties']['labels']['additionalProperties'].update(minLength=2, maxLength=6)
schema['properties']['trees'] = copy.deepcopy(d['components']['schemas']['Holder']['properties']['trees'])
schema['properties']['rootTree'] = copy.deepcopy(d['components']['schemas']['Holder']['properties']['rootTree'])
schema['properties']['borrowedWord'] = copy.deepcopy(d['components']['schemas']['Holder']['properties']['borrowedWord'])
schema['properties']['borrowedTree'] = copy.deepcopy(d['components']['schemas']['Holder']['properties']['borrowedTree'])
for name in ['rootWord', 'rootWords', 'rootWordTree', 'rootWordUnannotated', 'rootWordTreeAlias', 'rootWordTreeUnannotated', 'nativeFactoryUnannotated']:
    schema['properties'][name] = copy.deepcopy(d['components']['schemas']['Holder']['properties'][name])
bounds = copy.deepcopy(schema)
if checked:
    bounds['properties']['codes']['items']['minLength'] = 3
else:
    bounds['properties']['codes']['items']['enum'].append('separate')
other_default = copy.deepcopy(schema)
other_default['properties']['codes']['default'] = ['two']
def operation(name, shape):
    return {'get': {'operationId': name, 'responses': {'200': {'content': {'application/json': {'schema': shape}}}}}}
paths = {f'/{name}': operation(name, shape) for name, shape in [('alpha', schema), ('beta', schema), ('bounds', bounds), ('defaults', other_default)]}
seed = {'properties': {'state': copy.deepcopy(schema['properties']['state'])}, 'required': ['state'], 'default': {'state': 'one'}}
components = {'Seed': seed, 'SeedAlias': {'$ref': '#/components/schemas/Seed', 'default': {'state': 'one'}}}
components.update({name: copy.deepcopy(d['components']['schemas'][name]) for name in ['BoundedTree', 'BoundedTreeAlias', 'MaybeBoundedTree']})
components.update({'BorrowedOwner': {'properties': {'word': copy.deepcopy(d['components']['schemas']['Holder']['properties']['state']), 'tree': copy.deepcopy(d['components']['schemas']['Holder']['properties']['borrowableTree'])}}, 'BorrowedWord': {'$ref': '#/components/schemas/BorrowedOwner/properties/word', 'default': 'two'}, 'MaybeBorrowedWord': {'$ref': '#/components/schemas/BorrowedWord', 'nullable': True, 'default': None}, 'BorrowedTree': {'$ref': '#/components/schemas/BorrowedOwner/properties/tree', 'default': {'factory': {'a': 'two'}}}, 'BorrowedTreeAlias': {'$ref': '#/components/schemas/BorrowedTree', 'default': {'alias': {'a': 'one'}}}, 'MaybeBorrowedTree': {'$ref': '#/components/schemas/BorrowedTreeAlias', 'nullable': True, 'default': None}})
components.update({name: copy.deepcopy(d['components']['schemas'][name]) for name in ['RootWord', 'RootWordAlias', 'MaybeRootWord', 'NullRootWord', 'RootWordTree']})
factory_names = ['RootWordTreeAlias', 'MaybeRootWordTree', 'RootWordTreeHolder', 'FactoryInt', 'FactoryIntAlias', 'FactoryIntRows', 'FactoryIntRowsAlias', 'MaybeFactoryIntRows', 'FactoryLong', 'FactoryLongAlias', 'FactoryLongRows', 'FactoryLongRowsAlias', 'MaybeFactoryLongRows', 'FactoryFloat', 'FactoryFloatAlias', 'FactoryFloatRows', 'FactoryFloatRowsAlias', 'MaybeFactoryFloatRows', 'FactoryDouble', 'FactoryDoubleAlias', 'FactoryDoubleRows', 'FactoryDoubleRowsAlias', 'MaybeFactoryDoubleRows', 'FactoryBoolean', 'FactoryBooleanAlias', 'FactoryBooleanRows', 'FactoryBooleanRowsAlias', 'MaybeFactoryBooleanRows']
components.update({name: copy.deepcopy(d['components']['schemas'][name]) for name in factory_names})
(p/'options.json').write_text(json.dumps({'openapi': '3.1.0', 'paths': paths, 'components': {'schemas': components}}))
(p/'dedup.json').write_text(json.dumps({'openapi': '3.1.0', 'paths': paths, 'components': {'schemas': components}}))
(p/'merge.json').write_text(json.dumps({'openapi': '3.1.0', 'paths': paths, 'components': {'schemas': {**components, 'Alpha': schema, 'Zeta': schema}}}))
# Independent shallow width and single-field depth probes retain existing caps/heap.
wide = {'properties': {f'f{i}': {'type': 'array', 'items': {'enum': ['one'], **({'minLength': 2, 'maxLength': 6, 'pattern': '^[a-z]+$'} if checked else {})}} for i in range(240)}}
wide_scalar = {'properties': {f'f{i}': {'type': 'string', 'format': 'date', 'enum': ['one'], 'default': 'one', **({'minLength': 2, 'maxLength': 6, 'pattern': '^[a-z]+$'} if checked else {})} for i in range(240)}}
wide_nested = {'properties': {f'f{i}': {'type': 'array', 'items': {'additionalProperties': copy.deepcopy(schema['properties']['codes']['items'])}} for i in range(240)}}
deep = copy.deepcopy(schema['properties']['codes']['items'])
for _ in range(16):
    deep = {'type': 'array', 'items': deep}
(p/'limits.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': {'Wide': wide, 'WideScalars': wide_scalar, 'WideNested': wide_nested, 'DeepNested': {'properties': {'value': deep}}, **{name: copy.deepcopy(components[name]) for name in factory_names}, **{name: copy.deepcopy(components[name]) for name in ['BoundedTree', 'BoundedTreeAlias', 'MaybeBoundedTree', 'BorrowedOwner', 'BorrowedWord', 'MaybeBorrowedWord', 'BorrowedTree', 'BorrowedTreeAlias', 'MaybeBorrowedTree', 'RootWord', 'RootWordAlias', 'MaybeRootWord', 'NullRootWord', 'RootWordTree']}, 'WideBorrowed': {'properties': {f'f{i}': {'$ref': '#/components/schemas/BorrowedWord'} for i in range(240)}}, 'WideRootWords': {'properties': {f'f{i}': {'$ref': '#/components/schemas/RootWordAlias', 'default': 'one'} for i in range(240)}}, 'WideRootTrees': {'properties': {f'f{i}': {'$ref': '#/components/schemas/BoundedTreeAlias'} for i in range(240)}}}}}))
copy_calls = ''
if sys.argv[2] == 'data_class':
    copy_calls = '''
 val value = org.example.stringenumbounds.models.HolderDto(required = listOf(org.example.stringenumbounds.models.HolderDto.RequiredInner.TWO))
 check(value.copy(kotlin = "copy").labels == value.labels)
 check(value.copy(codes = listOf(org.example.stringenumbounds.models.HolderDto.CodesInner.Unrecognized("future"))).codes?.single()?.serialName == "future")
 check(value.copy(state = org.example.stringenumbounds.models.HolderDto.State.Unrecognized("future")).state?.serialName == "future")
 check(value.copy(trees = listOf(mapOf("new" to org.example.stringenumbounds.models.HolderDto.TreesInnerValue.Unrecognized("future")))).trees?.single()?.get("new")?.serialName == "future")
 check(value.copy(borrowedWord = org.example.stringenumbounds.models.HolderDto.State.Unrecognized("future")).borrowedWord?.serialName == "future")
'''
    if checked:
        copy_calls += '''
 for ((bad, rule) in listOf("x" to "minLength", "toolong" to "maxLength", "BAD" to "pattern", "!" to "minLength", "futurelong" to "maxLength", "FUTURE" to "pattern")) {
  val failure = runCatching { value.copy(codes = listOf(org.example.stringenumbounds.models.HolderDto.CodesInner.fromSerialName(bad))) }.exceptionOrNull()
  check(failure is IllegalArgumentException && failure.message.orEmpty().contains(rule))
  check(runCatching { value.copy(labels = mapOf("a" to org.example.stringenumbounds.models.HolderDto.LabelsValue.fromSerialName(bad))) }.exceptionOrNull() is IllegalArgumentException)
  check(runCatching { value.copy(state = org.example.stringenumbounds.models.HolderDto.State.fromSerialName(bad)) }.exceptionOrNull() is IllegalArgumentException)
 }
 check(runCatching { value.copy(trees = listOf(mapOf("bad" to org.example.stringenumbounds.models.HolderDto.TreesInnerValue.X))) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(borrowedWord = org.example.stringenumbounds.models.HolderDto.State.X) }.exceptionOrNull() is IllegalArgumentException)
 check(runCatching { value.copy(required = emptyList()) }.exceptionOrNull() is IllegalArgumentException)
'''
main = '''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val optimized = org.example.stringboundsoptions.models.ResponseDto()
 check(org.example.stringboundsoptions.models.`RootWordDto default`().serialName == "one")
 check(org.example.stringboundsoptions.models.`RootWordAliasDto default`().serialName == "two")
 check(org.example.stringboundsoptions.models.`MaybeRootWordDto default`() == null && org.example.stringboundsoptions.models.`NullRootWordDto default`() == null)
 check(org.example.stringboundsdedup.models.`RootWordDto default`().serialName == "one")
 check(org.example.stringboundsdedup.models.`RootWordAliasDto default`().serialName == "two")
 check(org.example.stringboundsdedup.models.`MaybeRootWordDto default`() == null && org.example.stringboundsdedup.models.`NullRootWordDto default`() == null)
 check(org.example.stringboundsmerge.models.`RootWordDto default`().serialName == "one")
 check(org.example.stringboundsmerge.models.`RootWordAliasDto default`().serialName == "two")
 check(org.example.stringboundsmerge.models.`MaybeRootWordDto default`() == null && org.example.stringboundsmerge.models.`NullRootWordDto default`() == null)
 check(org.example.stringboundsoptions.models.`BorrowedTreeDto default`()["factory"]?.get("a")?.serialName == "two")
 check(org.example.stringboundsoptions.models.`BorrowedTreeAliasDto default`()["alias"]?.get("a")?.serialName == "one")
 check(org.example.stringboundsoptions.models.`MaybeBorrowedTreeDto default`() == null)
 check(org.example.stringboundsdedup.models.`BorrowedTreeDto default`()["factory"]?.get("a")?.serialName == "two")
 check(org.example.stringboundsdedup.models.`BorrowedTreeAliasDto default`()["alias"]?.get("a")?.serialName == "one")
 check(org.example.stringboundsdedup.models.`MaybeBorrowedTreeDto default`() == null)
 check(org.example.stringboundsmerge.models.`BorrowedTreeDto default`()["factory"]?.get("a")?.serialName == "two")
 check(org.example.stringboundsmerge.models.`BorrowedTreeAliasDto default`()["alias"]?.get("a")?.serialName == "one")
 check(org.example.stringboundsmerge.models.`MaybeBorrowedTreeDto default`() == null)
 check(optimized.codes?.single() == org.example.stringboundsoptions.models.ResponseDto.CodesInner.ONE)
 check(optimized.labels?.get("a") == org.example.stringboundsoptions.models.ResponseDto.LabelsValue.YES)
 check(optimized.state?.serialName == "one")
 check(optimized.trees?.get(1)?.get("a/~")?.serialName == "one")
 check(optimized.rootTree?.single()?.get("own")?.serialName == "one")
 check(optimized.borrowedWord?.serialName == "one" && optimized.borrowedTree?.get("field")?.get("a")?.serialName == "one")
 check(optimized.owned?.state?.serialName == "two" && optimized.ownedRows?.single()?.state?.serialName == "two")
 check(org.example.stringboundsoptions.api.models.AlphaResponseDto() == optimized)
 check(org.example.stringboundsoptions.api.models.BetaResponseDto() == optimized)
 check(org.example.stringboundsoptions.models.ResponseDto.CodesInner.serializer().descriptor.serialName == "org.example.stringboundsoptions.models.ResponseDto.CodesInner")
 val dedup = org.example.stringboundsdedup.api.models.BetaResponseDto()
 check(dedup.codes?.single() == org.example.stringboundsdedup.api.models.AlphaResponseDto.CodesInner.ONE)
 val merged = org.example.stringboundsmerge.models.ZetaDto()
 check(merged.labels?.get("a") == org.example.stringboundsmerge.models.AlphaDto.LabelsValue.YES)
 check(merged.trees?.get(1)?.get("a/~")?.serialName == "one")
 check(dedup.trees?.get(1)?.get("a/~")?.serialName == "one")
 check(org.example.stringboundsmerge.api.models.AlphaResponseDto() == merged)
 check(org.example.stringboundsdedup.api.models.DefaultsResponseDto().codes?.single()?.serialName == "two")
 check(org.example.stringboundsoptions.api.models.DefaultsResponseDto().codes?.single()?.serialName == "two")
 check(org.example.stringboundsoptions.api.models.BoundsResponseDto.CodesInner.ONE.serialName == "one")
 check(org.example.stringboundsdedup.api.models.BoundsResponseDto.CodesInner.ONE.serialName == "one")
 check(org.example.stringboundsmerge.api.models.BoundsResponseDto.CodesInner.ONE.serialName == "one")
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
  val json = Json { encodeDefaults = defaults; explicitNulls = nulls }
  check(json.decodeFromString(org.example.stringboundsoptions.models.ResponseDto.serializer(), "{}") == optimized)
  check(json.decodeFromString(org.example.stringboundsdedup.api.models.AlphaResponseDto.serializer(), "{}") == dedup)
  check(json.decodeFromString(org.example.stringboundsmerge.models.AlphaDto.serializer(), "{}") == merged)
  check(json.decodeFromString(org.example.stringboundsoptions.models.ResponseDto.serializer(), """{"codes":["future"]}""").codes?.single()?.serialName == "future")
'''
if checked:
    main += '''
  for (bad in listOf("x", "FUTURE", "futurelong")) {
   check(runCatching { json.decodeFromString(org.example.stringboundsoptions.models.ResponseDto.serializer(), "{\\"codes\\":[\\"$bad\\"]}") }.isFailure)
   check(runCatching { json.decodeFromString(org.example.stringboundsdedup.api.models.AlphaResponseDto.serializer(), "{\\"codes\\":[\\"$bad\\"]}") }.isFailure)
   check(runCatching { json.decodeFromString(org.example.stringboundsmerge.models.AlphaDto.serializer(), "{\\"codes\\":[\\"$bad\\"]}") }.isFailure)
  }
'''
main += ' }\n' + copy_calls + ' stringEnumCollectionBoundsContracts(' + sys.argv[3] + ') }\n'
factory_checks = []
for package in ['org.example.stringboundsoptions', 'org.example.stringboundsdedup', 'org.example.stringboundsmerge']:
    factory_checks += [f' check({package}.models.`RootWordTreeDto default`()["factory"]?.single()?.serialName == "one")', f' check({package}.models.`RootWordTreeAliasDto default`()["alias"]?.single()?.serialName == "two")', f' check({package}.models.`MaybeRootWordTreeDto default`() == null)']
    for name, first, second in [('Int', '1', '2'), ('Long', '1L', '2L'), ('Float', '1.0f', '2.0f'), ('Double', '1.0', '2.0'), ('Boolean', 'true', 'false')]:
        factory_checks += [f' check({package}.models.`Factory{name}Dto default`() == {first})', f' check({package}.models.`Factory{name}AliasDto default`() == {second})', f' check({package}.models.`Factory{name}RowsDto default`() == listOf({first}))', f' check({package}.models.`Factory{name}RowsAliasDto default`() == listOf({second}))', f' check({package}.models.`MaybeFactory{name}RowsDto default`() == null)']
main = main.replace(' val optimized =', '\n'.join(factory_checks) + '\n val optimized =', 1)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(main)
b = p/'build.gradle.kts'
b.write_text(b.read_text() + '''
tasks.register<JavaExec>("runStringEnumCollectionBoundsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.stringenumbounds --api-client-name StringEnumBounds --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.stringboundsoptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    # Source with bounds must reject off and preserve the accepted output.
    "$workspace/openapi-kt" generate --input "$project/bounded.json" --output "$project/build/generated" --package-name org.example.stringenumbounds --diagnostic-format json > "$project/rejection.json" && exit 1
    python3 - "$project/rejection.json" <<'PYREJECT'
import json
import sys
d = json.load(open(sys.argv[1]))
assert d['status'] == 'error' and d['files'] == []
assert d['diagnostics'][0]['code'] == 'E_SCHEMA_UNSUPPORTED'
PYREJECT
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    for mode in dedup merge limits; do
      optimization=()
      if [[ "$mode" == dedup ]]; then optimization=(--deduplicate-models); fi
      if [[ "$mode" == merge ]]; then optimization=(--merge-identical-models); fi
      if [[ "$checked" == true ]]; then optimization+=(--validation); fi
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "org.example.stringbounds$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "$prepare_only" == false ]]; then
      "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runStringEnumCollectionBoundsContracts
      echo "FRESH STRING ENUM COLLECTION BOUNDS $style checked=$checked PASS: $project"
    fi
  done
done
printf 'Fresh String enum collection bounds projects (prepare_only=%s): %s\n' "$prepare_only" "$workspace"
