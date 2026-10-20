#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/referenced-bytes-field-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in false true; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferencedBytesFieldDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    python3 - "$project" "$style" "$checked" <<'PY'
from pathlib import Path
import json, sys
p = Path(sys.argv[1]); checked = sys.argv[3] == 'true'; ref = '#/components/schemas/'
d = json.loads(Path('tests/fixtures/http/referenced-bytes-field-defaults.json').read_text())
if checked: d['components']['schemas']['Capped']['minItems'] = 1
(p/'input.json').write_text(json.dumps(d))
model = {'properties': {'value': {'$ref': ref+'Bytes', 'default': [-128,127]}}}
paths = {f'/{n}': {'get': {'operationId': n, 'responses': {'200': {'content': {'application/json': {'schema': model}}}}}} for n in ['alpha', 'beta']}
for mode in ['options', 'dedup', 'merge']:
    schemas = {'Bytes': {'type': 'string', 'format': 'byte', 'nullable': True}}
    if mode == 'merge': schemas.update(Alpha=model, Zeta=model, Different={'properties': {'value': {'$ref': ref+'Bytes', 'default': [2]}}})
    (p/f'{mode}.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': schemas}, 'paths': paths}))
# Independent width/source/check/default boundaries. No combined limit claim.
schemas = {'Bytes': {'type': 'string', 'format': 'byte', 'nullable': True}, 'Hops': {'properties': {}}}
schemas['Wide'] = {'properties': {f'v{i}': {'$ref': ref+'Bytes', 'default': [1]} for i in range(240)}}
for i in range(46):
    schemas['Hops']['properties'][f'h{i}'] = {'$ref': ref+('Bytes' if i == 0 else f'Hops/properties/h{i-1}'), 'nullable': False}
schemas['Source48'] = {'properties': {'value': {'$ref': ref+'Hops/properties/h45', 'default': [1]}}}
schemas['Nodes'] = {'properties': {'value': {'$ref': ref+'Bytes', 'default': [0]*16383}}}
schemas['BytesLimit'] = {'properties': {'value': {'$ref': ref+'Bytes', 'default': [-128]*6553+[0]}}}
leaf = {'type': 'string', 'format': 'byte'}
for _ in range(16): leaf = {'type': 'array', 'items': leaf, **({'minItems': 0} if checked else {})}
schemas['Deep'] = leaf
schemas['DeepHolder'] = {'properties': {'value': {'$ref': ref+'Deep', 'default': []}}}
# Compact JSON keeps the independent 32KiB original annotation budget exact.
(p/'resources.json').write_text(json.dumps({'openapi': '3.1.0', 'components': {'schemas': schemas}}, separators=(',', ':')))
copy = '' if sys.argv[2] == 'plain_class' else '''
 val value = org.example.refbytes.models.HolderDto(requiredBlob = byteArrayOf(9))
 val copied = value.copy(kotlin = "copy", kotlinx = "copy")
 check(copied.blob === value.blob)
 check(value.copy(blob = null).blob == null)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
fun main() = runBlocking {
 val optimized = org.example.refbytesoptions.models.ResponseDto()
 val actual: org.example.refbytesoptions.models.BytesDto = optimized.value
 val builtin: ByteArray? = actual
 check(builtin.contentEquals(byteArrayOf(-128,127)))
 check(org.example.refbytesoptions.api.models.BetaResponseDto()::class == optimized::class)
 check(org.example.refbytesoptions.models.ResponseDto.serializer().descriptor.serialName == "org.example.refbytesoptions.models.ResponseDto")
 val dedup = org.example.refbytesdedup.api.models.BetaResponseDto()
 check(dedup.value.contentEquals(byteArrayOf(-128,127)) && dedup::class == org.example.refbytesdedup.api.models.AlphaResponseDto()::class)
 check(org.example.refbytesdedup.api.models.BetaResponseDto.serializer().descriptor.serialName == "org.example.refbytesdedup.api.models.AlphaResponseDto")
 val merged = org.example.refbytesmerge.models.ZetaDto()
 check(merged.value.contentEquals(byteArrayOf(-128,127)) && org.example.refbytesmerge.api.models.AlphaResponseDto()::class == merged::class)
 check(org.example.refbytesmerge.models.ZetaDto.serializer().descriptor.serialName == "org.example.refbytesmerge.models.AlphaDto")
 check(org.example.refbytesmerge.models.DifferentDto().value.contentEquals(byteArrayOf(2)))
 for (defaults in listOf(false, true)) for (nulls in listOf(false, true)) {
   val codec = Json { encodeDefaults = defaults; explicitNulls = nulls }
   check(codec.decodeFromString(org.example.refbytesoptions.models.ResponseDto.serializer(), "{}").value.contentEquals(byteArrayOf(-128,127)))
   check(codec.decodeFromString(org.example.refbytesdedup.api.models.AlphaResponseDto.serializer(), "{}").value.contentEquals(byteArrayOf(-128,127)))
   check(codec.decodeFromString(org.example.refbytesmerge.models.AlphaDto.serializer(), "{}").value.contentEquals(byteArrayOf(-128,127)))
 }
 check(org.example.refbytesresources.models.WideDto().v239.contentEquals(byteArrayOf(1)))
 check(org.example.refbytesresources.models.Source48Dto().value.contentEquals(byteArrayOf(1)))
 val nodes = org.example.refbytesresources.models.NodesDto()
 check(nodes.value!!.size == 16383 && nodes.value !== org.example.refbytesresources.models.NodesDto().value)
 val bytes = org.example.refbytesresources.models.BytesLimitDto()
 check(bytes.value!!.size == 6554 && bytes.value!!.first() == (-128).toByte() && bytes.value!!.last() == 0.toByte())
 check(bytes.value !== org.example.refbytesresources.models.BytesLimitDto().value)
 check(org.example.refbytesresources.models.DeepHolderDto().value!!.isEmpty())
''' + copy + ' referencedBytesFieldDefaultsContracts(' + sys.argv[3] + ') }\n')
b = p/'build.gradle.kts'; b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runReferencedBytesFieldDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
    flags=(--input "$project/input.json" --output "$project/build/generated" --package-name org.example.refbytes --api-client-name ReferencedBytes --model-name-suffix Dto --model-style "$style")
    options=(--input "$project/options.json" --output "$project/build/generated-options" --package-name org.example.refbytesoptions --model-name-suffix Dto --model-style "$style" --hoist-shared-shapes --deduplicate-models --merge-identical-models --lock-hoisted-names)
    if [[ "$checked" == true ]]; then flags+=(--validation); options+=(--validation); fi
    "$workspace/openapi-kt" generate "${flags[@]}"
    "$workspace/openapi-kt" generate "${flags[@]}" --check
    "$workspace/openapi-kt" generate "${options[@]}"
    "$workspace/openapi-kt" generate "${options[@]}" --check
    for mode in dedup merge resources; do
      optimization=()
      if [[ "$mode" == dedup ]]; then optimization=(--deduplicate-models); fi
      if [[ "$mode" == merge ]]; then optimization=(--merge-identical-models); fi
      if [[ "$checked" == true ]]; then optimization+=(--validation); fi
      mode_flags=(--input "$project/$mode.json" --output "$project/build/generated-limits/$mode" --package-name "org.example.refbytes$mode" --model-name-suffix Dto --model-style "$style" "${optimization[@]}")
      "$workspace/openapi-kt" generate "${mode_flags[@]}"
      "$workspace/openapi-kt" generate "${mode_flags[@]}" --check
    done
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED REFERENCED BYTES FIELD DEFAULTS $style checked=$checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferencedBytesFieldDefaultsContracts
    echo "FRESH REFERENCED BYTES FIELD DEFAULTS $style checked=$checked PASS: $project"
  done
done
