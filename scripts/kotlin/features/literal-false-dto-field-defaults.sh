#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/literal-false-dto-field-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/LiteralFalseDTOFieldDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  validation=(); [[ "$checked" == off ]] || validation+=(--validation)
  for mode in base optimized; do
   package=org.example.literalfielddefaults; output="$project/build/generated"; suffix=; options=()
   if [[ "$mode" == optimized ]]; then
    package+=optimized; output="$project/build/generated-limits"; suffix=Dto
    options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
   fi
   args=(--input tests/fixtures/http/literal-false-dto-field-defaults.json --output "$output" --package-name "$package" --api-client-name LiteralFieldDefaultsClient --model-name-suffix "$suffix" --model-style "$style")
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
  done
  python3 - "$project" "$checked" <<'PYWIDE'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on';ref=lambda name,**extra:dict({'$ref':'#/components/schemas/'+name},**extra)
props={f'p{i:03d}':False for i in range(239)};props['labels']={'type':'array','items':{'type':'string'}}
target={'properties':props,'default':{'labels':['target']}}
if checked:target.update(minProperties=0,maxProperties=1)
fields={f'p{i:03d}':ref('Target',default={'labels':['field']}) for i in range(240)}
schemas={'Target':target,'Wide':{'properties':fields},'WideRequired':{'required':list(fields),'properties':fields}}
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}}))
large={'Target':{'properties':{'blocked':False,'labels':{'type':'array','items':{'type':'string','nullable':True}}}},'Consumer':{'properties':{'value':ref('Target',default={'labels':[None]*6551})}}}
(p/'large.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':large}}))
PYWIDE
  for mode in wide large; do
   output="$project/build/generated-options"; [[ "$mode" == wide ]] || output="$project/build/generated-large"
   args=(--input "$project/$mode.json" --output "$output" --package-name "org.example.literalfielddefaults$mode" --model-name-suffix Dto --model-style "$style")
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}"
   "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" --check
  done
  python3 - "$project" <<'PYMAIN'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);consumer=(p/'src/main/kotlin/org/example/contracts/LiteralFalseDTOFieldDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.literalfielddefaults.LiteralFieldDefaultsClient','org.example.literalfielddefaultsoptimized.LiteralFieldDefaultsClient')
for name in json.loads(Path('tests/fixtures/http/literal-false-dto-field-defaults.json').read_text())['components']['schemas']:
 optimized=optimized.replace('org.example.literalfielddefaults.models.'+name+' as ','org.example.literalfielddefaultsoptimized.models.'+name+'Dto as ')
(opt/'LiteralFalseDTOFieldDefaultsContracts.kt').write_text(optimized)
assert json.loads((p/'build/generated-limits/.openapi-kt-names.json').read_text())['entries'],'unrelated ordinary request shapes retain nonempty name locks'
wide=r'''
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
fun wideLiteralFalseDTOFieldDefaults() {
 val first=org.example.literalfielddefaultswide.models.WideDto()
 val next=org.example.literalfielddefaultswide.models.WideDto()
 check(first.p000?.labels==listOf("field") && first.p239?.labels==listOf("field"))
 check(first.p000 !== next.p000 && first.p000?.labels !== next.p000?.labels)
 check(org.example.literalfielddefaultswide.models.TargetDto.default().labels==listOf("target"))
 val optional=org.example.literalfielddefaultswide.models.WideDto.serializer().descriptor
 val required=org.example.literalfielddefaultswide.models.WideRequiredDto.serializer().descriptor
 check(optional.elementsCount==240 && required.elementsCount==240)
 for(index in 0 until 240) {check(optional.isElementOptional(index));check(!required.isElementOptional(index))}
 val value=constructWide(org.example.literalfielddefaultswide.models.TargetDto.default())
 check(value.p239.labels==listOf("target"))
 val large=org.example.literalfielddefaultslarge.models.ConsumerDto()
 val second=org.example.literalfielddefaultslarge.models.ConsumerDto()
 check(large.value?.labels?.size==6551 && large.value?.labels !== second.value?.labels)
 for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
  val codec=kotlinx.serialization.json.Json {encodeDefaults=defaults;explicitNulls=nulls}
  check(codec.decodeFromString<org.example.literalfielddefaultswide.models.WideDto>(codec.encodeToString(first)).p239?.labels==listOf("field"))
  check(codec.decodeFromString<org.example.literalfielddefaultswide.models.WideRequiredDto>(codec.encodeToString(value)).p239.labels==listOf("target"))
  check(runCatching {codec.decodeFromString<org.example.literalfielddefaultswide.models.WideDto>("{\"p239\":{\"p238\":null}}")}.isFailure)
 }
}
'''
construct='private fun constructWide(value: org.example.literalfielddefaultswide.models.TargetDto) = org.example.literalfielddefaultswide.models.WideRequiredDto('+','.join(['value']*240)+')\n'
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text('package org.example.contracts\nimport kotlinx.serialization.encodeToString\nimport kotlinx.serialization.decodeFromString\n'+construct+wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main()=runBlocking {wideLiteralFalseDTOFieldDefaults();literalFalseDTOFieldDefaultsContracts();org.example.optimizedcontracts.literalFalseDTOFieldDefaultsContracts()}\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
kotlin.sourceSets.main {kotlin.srcDir(layout.buildDirectory.dir("generated-large"))}
tasks.register<JavaExec>("runLiteralFalseDTOFieldDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
   echo "PREPARED LITERAL FALSE DTO FIELD DEFAULTS $style $checked: $project"
   continue
  fi
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runLiteralFalseDTOFieldDefaultsContracts
  echo "FRESH LITERAL FALSE DTO FIELD DEFAULTS PROFILE $style-$checked PASS: $project"
 done
done
