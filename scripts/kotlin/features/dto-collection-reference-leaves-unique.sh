#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/dto-collection-reference-leaves-unique-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DTOCollectionReferenceLeavesUniqueContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(Path('tests/fixtures/http/dto-collection-reference-leaves-unique.json').read_text())
if sys.argv[2]=='off':
 for name in ['Holder','NullableHolder','InlineHolder']:d['components']['schemas'][name]['properties']['values'].pop('uniqueItems')
 d['components']['schemas']['Item']['properties']['numbers']['items'].pop('minimum')
 d['components']['schemas']['RefInt'].pop('minimum')
 d['components']['schemas']['RefFloats']['items'].pop('minimum')
 # The established unchecked borrowed-native gate stays closed.
 # The off control retains its primitive borrowed collection; checked keeps enum.
 d['components']['schemas']['NativeBorrowOwner']['properties']['e/nums']['items'].pop('enum')
 d['components']['schemas']['LeafRefFloat'].pop('enum')
 d['components']['schemas']['LeafRefOwner']['properties']['amount'].pop('minimum')
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
wide={'openapi':'3.1.0','components':{'schemas':{'Item':{'properties':{'p'+str(i):{'$ref':'#/components/schemas/Value'} for i in range(240)}},'Holder':{'properties':{'values':{'type':'array','items':{'$ref':'#/components/schemas/Item'}}}}}}}
wide['components']['schemas']['Scalar']={'type':'string','enum':['ready','done']}
wide['components']['schemas']['Value']={'type':'array','items':{'$ref':'#/components/schemas/Scalar'}}
if sys.argv[2]=='on':wide['components']['schemas']['Holder']['properties']['values']['uniqueItems']=True
wide['components']['schemas']['MegaHolder']={'properties':{'v'+str(i):json.loads(json.dumps(wide['components']['schemas']['Holder']['properties']['values'])) for i in range(240)}}

(p/'wide.json').write_text(json.dumps(wide,indent=2)+'\n')

field={'$ref':'#/components/schemas/Scalar'}
expr='org.example.dtocollectionreferenceleavesuniquedeep.models.ScalarDto.READY'
for _ in range(15):
 field={'type':'array','items':field}
 expr='listOf('+expr+')'
deep={'openapi':'3.1.0','components':{'schemas':{'Scalar':{'type':'string','enum':['ready','done']},'Value':field,'Item':{'properties':{'id':{'type':'string'},'values':{'$ref':'#/components/schemas/Value'}},'required':['id']},'Holder':{'properties':{'values':{'type':'array','items':{'$ref':'#/components/schemas/Item'}}}}}}}
if sys.argv[2]=='on':deep['components']['schemas']['Holder']['properties']['values']['uniqueItems']=True
(p/'deep.json').write_text(json.dumps(deep,indent=2)+'\n')
deep_checks=' val deepItem=org.example.dtocollectionreferenceleavesuniquedeep.models.ItemDto(id="deep",values='+expr+'); val deepHolder=org.example.dtocollectionreferenceleavesuniquedeep.models.HolderDto(values=listOf(deepItem)); for(de in listOf(false,true)) for(en in listOf(false,true)) { val codec=kotlinx.serialization.json.Json {encodeDefaults=de;explicitNulls=en}; check(codec.decodeFromString<org.example.dtocollectionreferenceleavesuniquedeep.models.HolderDto>(codec.encodeToString(org.example.dtocollectionreferenceleavesuniquedeep.models.HolderDto.serializer(),deepHolder))==deepHolder) }; check(runCatching {org.example.dtocollectionreferenceleavesuniquedeep.models.HolderDto(values=listOf(deepItem,org.example.dtocollectionreferenceleavesuniquedeep.models.ItemDto(id="deep",values='+expr+')))}.isFailure=='+str(sys.argv[2]=='on').lower()+'); '

source=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DTOCollectionReferenceLeavesUniqueContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'DTOCollectionReferenceLeavesUniqueContracts.kt').write_text(source.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.dtocollectionreferenceleavesunique','org.example.dtocollectionreferenceleavesuniqueoptimized'))

(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking {'+deep_checks+' val wideItem=org.example.dtocollectionreferenceleavesuniquewide.models.ItemDto(p239=listOf(org.example.dtocollectionreferenceleavesuniquewide.models.ScalarDto.READY)); check(org.example.dtocollectionreferenceleavesuniquewide.models.HolderDto(values=listOf(wideItem)).values?.single()?.p239?.single()?.serialName=="ready"); check(org.example.dtocollectionreferenceleavesuniquewide.models.MegaHolderDto(v239=listOf(wideItem)).v239?.single()?.p239?.single()?.serialName=="ready"); dtoCollectionReferenceLeavesUniqueContracts('+str(sys.argv[2]=='on').lower()+'); org.example.optimizedcontracts.dtoCollectionReferenceLeavesUniqueContracts('+str(sys.argv[2]=='on').lower()+') }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runDTOCollectionReferenceLeavesUniqueContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
  flags=(--model-style "$style")
  [[ "$checked" == off ]] || flags+=(--validation)
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.dtocollectionreferenceleavesunique --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.dtocollectionreferenceleavesunique --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.dtocollectionreferenceleavesuniqueoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.dtocollectionreferenceleavesuniqueoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names --check
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.dtocollectionreferenceleavesuniquewide --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.dtocollectionreferenceleavesuniquewide --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input "$project/deep.json" --output "$project/build/generated-deep-object" --package-name org.example.dtocollectionreferenceleavesuniquedeep --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/deep.json" --output "$project/build/generated-deep-object" --package-name org.example.dtocollectionreferenceleavesuniquedeep --model-name-suffix Dto "${flags[@]}" --check
  python3 - "$project" <<'PYALIASES'
from pathlib import Path
import re,sys
project=Path(sys.argv[1]);package='org.example.dtocollectionreferenceleavesuniqueoptimized.models'
consumer=project/'src/main/kotlin/org/example/optimizedcontracts/DTOCollectionReferenceLeavesUniqueContracts.kt'
s=consumer.read_text()
aliases={}
for source in (project/'build/generated-limits'/Path(*package.split('.'))).glob('*.kt'):
 for match in re.finditer(r'^public typealias (\w+) = ([\w.]+)$',source.read_text(),re.MULTILINE):aliases[package+'.'+match[1]]=match[2]
for name in sorted(aliases):
 target=aliases[name]
 for _ in range(48):
  if target not in aliases:break
  target=aliases[target]
 s=s.replace(name+'.',target+'.')
consumer.write_text(s)
PYALIASES
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runDTOCollectionReferenceLeavesUniqueContracts
  echo "FRESH DTO COLLECTION REFERENCE LEAVES UNIQUE PROFILE $style-$checked PASS: $project"
 done
done
echo "Fresh DTO collection unique projects: $workspace"
