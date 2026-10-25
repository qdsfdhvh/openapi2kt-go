#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/literal-false-root-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for checked in off on; do
    project="$workspace/$style-$checked"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/LiteralFalseRootDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    validation=()
    [[ "$checked" == off ]] || validation+=(--validation)
    for mode in base optimized; do
      package=org.example.literalrootdefaults; output="$project/build/generated"; options=()
      if [[ "$mode" == optimized ]]; then
        package=org.example.literalrootdefaultsoptimized; output="$project/build/generated-limits"
        options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
      fi
      suffix=; [[ "$mode" == base ]] || suffix=Dto
      args=(--input tests/fixtures/http/literal-false-root-defaults.json --output "$output" --package-name "$package" --api-client-name LiteralRootDefaultsClient --model-name-suffix "$suffix" --model-style "$style")
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
      "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
    done
    python3 - "$project" "$checked" <<'PYWIDE'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on';props={f'p{i:03d}':False for i in range(240)}
schemas={'Wide':{'properties':props,'default':{}},'WideRequired':{'nullable':True,'required':list(props),'properties':props,'default':None}}
if checked: schemas['Wide']['maxProperties']=0;schemas['WideRequired']['minProperties']=240
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
values=[None]*6551
large={'Holder':{'properties':{'blocked':False,'labels':{'type':'array','items':{'type':'string','nullable':True}}},'default':{'labels':values}}}
(p/'large.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':large}}))
PYWIDE
    wide_args=(--input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.literalrootdefaultswide --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}"
    "$workspace/openapi-kt" generate "${wide_args[@]}" "${validation[@]}" --check
    large_args=(--input "$project/large.json" --output "$project/build/generated-large" --package-name org.example.literalrootdefaultslarge --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${large_args[@]}" "${validation[@]}"
    "$workspace/openapi-kt" generate "${large_args[@]}" "${validation[@]}" --check
    python3 - "$project" "$checked" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]); checked=str(sys.argv[2]=='on').lower()
consumer=(p/'src/main/kotlin/org/example/contracts/LiteralFalseRootDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
import json
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts')
optimized=optimized.replace('org.example.literalrootdefaults.LiteralRootDefaultsClient','org.example.literalrootdefaultsoptimized.LiteralRootDefaultsClient')
for name in json.loads(Path('tests/fixtures/http/literal-false-root-defaults.json').read_text())['components']['schemas']:
    optimized=optimized.replace('org.example.literalrootdefaults.models.'+name+' as ', 'org.example.literalrootdefaultsoptimized.models.'+name+'Dto as ')
(opt/'LiteralFalseRootDefaultsContracts.kt').write_text(optimized)
lock=json.loads((p/'build/generated-limits/.openapi-kt-names.json').read_text());assert lock['entries'],'unrelated eligible shapes must retain nonempty name locks'
wide = r"""
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
fun wideLiteralFalseFields() {
    val root=org.example.literalrootdefaultswide.models.WideDto.default()
    check(root.p000 == null && org.example.literalrootdefaultswide.models.WideDto.default() !== root)
    check(org.example.literalrootdefaultswide.models.WideRequiredDto.default() == null)
    val large=org.example.literalrootdefaultslarge.models.HolderDto.default()
    val next=org.example.literalrootdefaultslarge.models.HolderDto.default()
    check(large.labels?.size == 6551 && large.labels !== next.labels)
    check(large.labels!!.all { it == null })
    val optional=kotlinx.serialization.serializer<org.example.literalrootdefaultswide.models.WideDto>().descriptor
    val required=kotlinx.serialization.serializer<org.example.literalrootdefaultswide.models.WideRequiredDto>().descriptor
    check(optional.elementsCount == 240 && required.elementsCount == 240)
    for(index in 0 until 240) {
        check(optional.isElementOptional(index) && optional.getElementDescriptor(index).isNullable)
        check(!required.isElementOptional(index) && !required.getElementDescriptor(index).isNullable)
    }
    check(optional.getElementDescriptor(0).serialName != required.getElementDescriptor(0).serialName)
    val all=(0 until 240).joinToString(",","{","}") { "\"p"+it.toString().padStart(3,'0')+"\":null" }
    for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
        val json=kotlinx.serialization.json.Json { encodeDefaults=defaults; explicitNulls=nulls }
        val value=org.example.literalrootdefaultswide.models.WideDto()
        check(json.encodeToString(value) == "{}")
        check(json.encodeToString(json.decodeFromString<org.example.literalrootdefaultswide.models.WideDto>("{}")) == "{}")
        for(index in 0 until 240) {
            val field="p"+index.toString().padStart(3,'0')
            check(runCatching { json.decodeFromString<org.example.literalrootdefaultswide.models.WideDto>("{\"$field\":null}") }.isFailure)
        }
        check(runCatching { json.decodeFromString<org.example.literalrootdefaultswide.models.WideRequiredDto>("{}") }.isFailure)
        val failure=runCatching { json.decodeFromString<org.example.literalrootdefaultswide.models.WideRequiredDto>(all) }.exceptionOrNull()
        check(failure is kotlinx.serialization.SerializationException && failure.message == "Schema false admits no JSON value.")
    }
}
"""
construct='private fun constructWide(value: org.example.literalrootdefaultswide.models.WideRequiredDto.FalseValue) = org.example.literalrootdefaultswide.models.WideRequiredDto('+','.join(['value']*240)+')\n'
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text('package org.example.contracts\n'+construct+wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { wideLiteralFalseFields(); literalFalseRootDefaultsContracts(); org.example.optimizedcontracts.literalFalseRootDefaultsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
kotlin.sourceSets.main { kotlin.srcDir(layout.buildDirectory.dir("generated-large")) }

tasks.register<JavaExec>("runLiteralFalseRootDefaultsContracts") {
    dependsOn(tasks.named("classes"))
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
      echo "PREPARED LITERAL FALSE ROOT DEFAULTS $style $checked: $project"
      continue
    fi
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runLiteralFalseRootDefaultsContracts
    echo "FRESH LITERAL FALSE ROOT DEFAULTS PROFILE $style-$checked PASS: $project"
  done
done
echo "Fresh literal false root default projects: $workspace"
