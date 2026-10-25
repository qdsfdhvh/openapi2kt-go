#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/dto-native-reference-unique-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DTONativeReferenceUniqueContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(Path('tests/fixtures/http/dto-native-reference-unique.json').read_text())
if sys.argv[2]=='off':
 for name in ['Holder','NullableHolder','InlineHolder']:d['components']['schemas'][name]['properties']['values'].pop('uniqueItems')
 d['components']['schemas']['Item']['properties']['numbers']['items'].pop('minimum')
 d['components']['schemas']['RefInt'].pop('minimum')
 for name in ['RefInt','RefFloat','RefFlag']:d['components']['schemas'][name].pop('enum')
 d['components']['schemas']['BorrowOwner']['properties']['leaf/~'].pop('enum')
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
wide={'openapi':'3.1.0','components':{'schemas':{'Item':{'properties':{'p'+str(i):{'$ref':'#/components/schemas/Value'} for i in range(240)}},'Holder':{'properties':{'values':{'type':'array','items':{'$ref':'#/components/schemas/Item'}}}}}}}
wide['components']['schemas']['Value']={'type':'integer','enum':[0,1]}
if sys.argv[2]=='on':wide['components']['schemas']['Holder']['properties']['values']['uniqueItems']=True
wide['components']['schemas']['MegaHolder']={'properties':{'v'+str(i):json.loads(json.dumps(wide['components']['schemas']['Holder']['properties']['values'])) for i in range(240)}}

(p/'wide.json').write_text(json.dumps(wide,indent=2)+'\n')

field={'type':'string','enum':['ready','done']}
expr='org.example.dtonativereferenceuniquedeep.models.ItemDto.Values'+('Inner'*15)+'.READY'
for _ in range(15):
 field={'type':'array','items':field}
 expr='listOf('+expr+')'
deep={'openapi':'3.1.0','components':{'schemas':{'Item':{'properties':{'id':{'type':'string'},'values':field},'required':['id']},'Holder':{'properties':{'values':{'type':'array','items':{'$ref':'#/components/schemas/Item'}}}}}}}
if sys.argv[2]=='on':deep['components']['schemas']['Holder']['properties']['values']['uniqueItems']=True
(p/'deep.json').write_text(json.dumps(deep,indent=2)+'\n')
deep_checks=' val deepItem=org.example.dtonativereferenceuniquedeep.models.ItemDto(id="deep",values='+expr+'); val deepHolder=org.example.dtonativereferenceuniquedeep.models.HolderDto(values=listOf(deepItem)); for(de in listOf(false,true)) for(en in listOf(false,true)) { val codec=kotlinx.serialization.json.Json {encodeDefaults=de;explicitNulls=en}; check(codec.decodeFromString<org.example.dtonativereferenceuniquedeep.models.HolderDto>(codec.encodeToString(org.example.dtonativereferenceuniquedeep.models.HolderDto.serializer(),deepHolder))==deepHolder) }; check(runCatching {org.example.dtonativereferenceuniquedeep.models.HolderDto(values=listOf(deepItem,org.example.dtonativereferenceuniquedeep.models.ItemDto(id="deep",values='+expr+')))}.isFailure=='+str(sys.argv[2]=='on').lower()+'); '

source=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DTONativeReferenceUniqueContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'DTONativeReferenceUniqueContracts.kt').write_text(source.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.dtonativereferenceunique','org.example.dtonativereferenceuniqueoptimized'))

(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking {'+deep_checks+' check(org.example.dtonativereferenceuniquewide.models.HolderDto(values=listOf(org.example.dtonativereferenceuniquewide.models.ItemDto(p239=1))).values?.single()?.p239==1); check(org.example.dtonativereferenceuniquewide.models.MegaHolderDto(v239=listOf(org.example.dtonativereferenceuniquewide.models.ItemDto(p239=1))).v239?.single()?.p239==1); dtoNativeReferenceUniqueContracts('+str(sys.argv[2]=='on').lower()+'); org.example.optimizedcontracts.dtoNativeReferenceUniqueContracts('+str(sys.argv[2]=='on').lower()+') }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runDTONativeReferenceUniqueContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
  flags=(--model-style "$style")
  [[ "$checked" == off ]] || flags+=(--validation)
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.dtonativereferenceunique --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.dtonativereferenceunique --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.dtonativereferenceuniqueoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.dtonativereferenceuniqueoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names --check
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.dtonativereferenceuniquewide --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.dtonativereferenceuniquewide --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input "$project/deep.json" --output "$project/build/generated-deep-object" --package-name org.example.dtonativereferenceuniquedeep --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/deep.json" --output "$project/build/generated-deep-object" --package-name org.example.dtonativereferenceuniquedeep --model-name-suffix Dto "${flags[@]}" --check
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runDTONativeReferenceUniqueContracts
  echo "FRESH DTO NATIVE REFERENCE UNIQUE PROFILE $style-$checked PASS: $project"
 done
done
echo "Fresh DTO collection unique projects: $workspace"
