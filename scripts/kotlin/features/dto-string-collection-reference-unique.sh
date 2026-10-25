#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/dto-string-collection-reference-unique-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DTOStringCollectionReferenceUniqueContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(Path('tests/fixtures/http/dto-string-collection-reference-unique.json').read_text())
if sys.argv[2]=='off':
 for name in ['Holder','NullableHolder','InlineHolder']:d['components']['schemas'][name]['properties']['values'].pop('uniqueItems')
 d['components']['schemas']['Item']['properties']['numbers']['items'].pop('minimum')
 d['components']['schemas']['RefInt'].pop('minimum')
 d['components']['schemas']['RefFloats']['items'].pop('minimum')
 # The established unchecked borrowed-native gate stays closed.
 # The off control retains its primitive borrowed collection; checked keeps enum.
 d['components']['schemas']['NativeBorrowOwner']['properties']['e/nums']['items'].pop('enum')
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
wide={'openapi':'3.1.0','components':{'schemas':{'Item':{'properties':{'p'+str(i):{'$ref':'#/components/schemas/Value'} for i in range(240)}},'Holder':{'properties':{'values':{'type':'array','items':{'$ref':'#/components/schemas/Item'}}}}}}}
wide['components']['schemas']['Value']={'type':'array','items':{'type':'string','enum':['ready','done']}}
if sys.argv[2]=='on':wide['components']['schemas']['Holder']['properties']['values']['uniqueItems']=True
wide['components']['schemas']['MegaHolder']={'properties':{'v'+str(i):json.loads(json.dumps(wide['components']['schemas']['Holder']['properties']['values'])) for i in range(240)}}

(p/'wide.json').write_text(json.dumps(wide,indent=2)+'\n')

field={'type':'string','enum':['ready','done']}
expr='org.example.dtostringcollectionreferenceuniquedeep.models.ValueDto'+('Inner'*15)+'.READY'
for _ in range(15):
 field={'type':'array','items':field}
 expr='listOf('+expr+')'
deep={'openapi':'3.1.0','components':{'schemas':{'Value':field,'Item':{'properties':{'id':{'type':'string'},'values':{'$ref':'#/components/schemas/Value'}},'required':['id']},'Holder':{'properties':{'values':{'type':'array','items':{'$ref':'#/components/schemas/Item'}}}}}}}
if sys.argv[2]=='on':deep['components']['schemas']['Holder']['properties']['values']['uniqueItems']=True
(p/'deep.json').write_text(json.dumps(deep,indent=2)+'\n')
deep_checks=' val deepItem=org.example.dtostringcollectionreferenceuniquedeep.models.ItemDto(id="deep",values='+expr+'); val deepHolder=org.example.dtostringcollectionreferenceuniquedeep.models.HolderDto(values=listOf(deepItem)); for(de in listOf(false,true)) for(en in listOf(false,true)) { val codec=kotlinx.serialization.json.Json {encodeDefaults=de;explicitNulls=en}; check(codec.decodeFromString<org.example.dtostringcollectionreferenceuniquedeep.models.HolderDto>(codec.encodeToString(org.example.dtostringcollectionreferenceuniquedeep.models.HolderDto.serializer(),deepHolder))==deepHolder) }; check(runCatching {org.example.dtostringcollectionreferenceuniquedeep.models.HolderDto(values=listOf(deepItem,org.example.dtostringcollectionreferenceuniquedeep.models.ItemDto(id="deep",values='+expr+')))}.isFailure=='+str(sys.argv[2]=='on').lower()+'); '

source=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DTOStringCollectionReferenceUniqueContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'DTOStringCollectionReferenceUniqueContracts.kt').write_text(source.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.dtostringcollectionreferenceunique','org.example.dtostringcollectionreferenceuniqueoptimized'))

(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking {'+deep_checks+' val wideItem=org.example.dtostringcollectionreferenceuniquewide.models.ItemDto(p239=listOf(org.example.dtostringcollectionreferenceuniquewide.models.ValueDtoInner.READY)); check(org.example.dtostringcollectionreferenceuniquewide.models.HolderDto(values=listOf(wideItem)).values?.single()?.p239?.single()?.serialName=="ready"); check(org.example.dtostringcollectionreferenceuniquewide.models.MegaHolderDto(v239=listOf(wideItem)).v239?.single()?.p239?.single()?.serialName=="ready"); dtoStringCollectionReferenceUniqueContracts('+str(sys.argv[2]=='on').lower()+'); org.example.optimizedcontracts.dtoStringCollectionReferenceUniqueContracts('+str(sys.argv[2]=='on').lower()+') }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runDTOStringCollectionReferenceUniqueContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
  flags=(--model-style "$style")
  [[ "$checked" == off ]] || flags+=(--validation)
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.dtostringcollectionreferenceunique --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.dtostringcollectionreferenceunique --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.dtostringcollectionreferenceuniqueoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.dtostringcollectionreferenceuniqueoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names --check
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.dtostringcollectionreferenceuniquewide --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.dtostringcollectionreferenceuniquewide --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input "$project/deep.json" --output "$project/build/generated-deep-object" --package-name org.example.dtostringcollectionreferenceuniquedeep --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/deep.json" --output "$project/build/generated-deep-object" --package-name org.example.dtostringcollectionreferenceuniquedeep --model-name-suffix Dto "${flags[@]}" --check
  python3 - "$project" <<'PYALIASES'
from pathlib import Path
import re,sys
project=Path(sys.argv[1]);package='org.example.dtostringcollectionreferenceuniqueoptimized.models'
consumer=project/'src/main/kotlin/org/example/optimizedcontracts/DTOStringCollectionReferenceUniqueContracts.kt'
s=consumer.read_text()
for name in ['StringRefMapDtoValue','StringRefTreeDtoValueInner']:
 text=(project/'build/generated-limits'/Path(*package.split('.'))/(name+'.kt')).read_text()
 match=re.search(r'public typealias '+name+r' = ([\w.]+)',text)
 if match:s=s.replace(package+'.'+name,match[1])
consumer.write_text(s)
PYALIASES
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runDTOStringCollectionReferenceUniqueContracts
  echo "FRESH DTO STRING COLLECTION REFERENCE UNIQUE PROFILE $style-$checked PASS: $project"
 done
done
echo "Fresh DTO collection unique projects: $workspace"
