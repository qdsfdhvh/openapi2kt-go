#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/implicit-string-enum-null-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in false true; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  python3 - "$project" "$checked" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1])
cases=[
 ('string-enum-null-members','stringenumnullmembers','DefaultApi','StringEnumNullMemberContracts'),
 ('referenced-string-enum-null-member-defaults','refnullstrings','RefNullStrings','ReferencedStringEnumNullMemberDefaultsContracts'),
 ('nested-string-enum-collection-defaults','nestedstringenum','NestedStringEnum','NestedStringEnumCollectionDefaultsContracts'),
]
extra=[
 ('dto-allof-unique','dtoallofunique','DefaultClient','DtoAllOfUniqueContracts','dtoAllOfUniqueContracts(true)'),
 ('dto-collection-reference-leaves-unique','dtocollectionreferenceleavesunique','DefaultClient','DtoCollectionReferenceLeavesUniqueContracts','dtoCollectionReferenceLeavesUniqueContracts(true)'),
 ('dto-string-collection-reference-unique','dtostringcollectionreferenceunique','DefaultClient','DtoStringCollectionReferenceUniqueContracts','dtoStringCollectionReferenceUniqueContracts(true)'),
 ('nested-object-defaults','nestedobject','NestedObject','NestedObjectDefaultsContracts','nestedObjectDefaultsContracts(false)'),
 ('string-enum-collection-bounds','stringenumbounds','StringEnumBounds','StringEnumCollectionBoundsContracts','stringEnumCollectionBoundsContracts(true)'),
 ('string-enum-collection-defaults','stringenumcollections','StringEnumCollections','StringEnumCollectionDefaultsContracts','stringEnumCollectionDefaultsContracts(false)'),
 ('string-enum-unique','enumunique','StringEnumUniqueApiClient','StringEnumUniqueContracts','stringEnumUniqueContracts(true)'),
 ('tuple-enum-references','tupleenums','TupleEnumsApiClient','TupleEnumReferencesContracts','tupleEnumReferencesContracts(false)'),
]
def infer(v):
 if isinstance(v,dict):
  values=v.get('enum',[])
  if isinstance(values,list) and None in values and any(isinstance(x,str) for x in values):
   v.pop('type',None);v.pop('nullable',None)
  for key,child in v.items():
   if key not in {'enum','const','default'}:infer(child)
 elif isinstance(v,list):
  for child in v:infer(child)
selected=cases+([row[:4] for row in extra] if sys.argv[2]=='true' else [])
for fixture,package,client,consumer in selected:
 doc=json.loads(Path('tests/fixtures/http/'+fixture+'.json').read_text());infer(doc)
 (p/(fixture+'.json')).write_text(json.dumps(doc)+'\n')
 source=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/'+consumer+'.kt').read_text()
 (p/'src/main/kotlin/org/example/contracts'/ (consumer+'.kt')).write_text(source)
 if fixture in {row[0] for row in cases}:
  optimized=p/'src/main/kotlin/org/example/optimizedcontracts';optimized.mkdir(parents=True,exist_ok=True)
  (optimized/(consumer+'.kt')).write_text(source.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.'+package+'.','org.example.'+package+'optimized.'))
leaf={'enum':['on',None,'null']}
fields={f'p{i:03d}':{'$ref':'#/components/schemas/State'} for i in range(240)}
deep=leaf
for i in range(15):deep={'type':'array','items':deep}
schemas={'State':leaf,'FullState':{'enum':[None]+[f'v{i}' for i in range(239)]},'Wide':{'properties':fields,'required':list(fields)},'Deep':deep}
schemas['Root']={'oneOf':[{'$ref':'#/components/schemas/Leaf'}],'discriminator':{'propertyName':'kind','mapping':{'fixed':'Leaf'}}}
schemas['Leaf']={'properties':{'kind':{'enum':['fixed',None]}},'required':['kind']}
schemas['DefaultState']={'enum':['on',None],'default':'on'}
schemas['AliasDefault']={'$ref':'#/components/schemas/DefaultState','default':None}
for i in range(45):schemas[f'Alias{i}']={'$ref':'#/components/schemas/'+('State' if i==0 else f'Alias{i-1}')}
(p/'resources.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}})+'\n')
shape={'properties':{'value':{'enum':['on',None,'null'],'default':'on'}}}
paths={f'/{name}':{'get':{'operationId':name,'responses':{'200':{'content':{'application/json':{'schema':shape}}}}}} for name in ['alpha','beta']}
(p/'options.json').write_text(json.dumps({'openapi':'3.1.0','paths':paths})+'\n')
args=','.join(['null']*240)
calls='\n'.join(' '+prefix+call for prefix in ['', 'org.example.optimizedcontracts.'] for call in ['stringEnumNullMemberContracts()','referencedStringEnumNullMemberDefaultsContracts(false)','nestedStringEnumCollectionDefaultsContracts(false)'])
if sys.argv[2]=='true':calls+='\n'+'\n'.join(' '+row[4] for row in extra)
main='''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.Json
import kotlinx.serialization.serializer
import org.example.implicitstringresources.models.*
fun main()=runBlocking {
'''+calls+'''
 check(!StateDto.serializer().descriptor.isNullable)
 check(serializer<Alias44Dto>().descriptor.isNullable)
 check(Json.decodeFromString<Alias44Dto>("null")==null)
 check(`DefaultStateDto default`()===DefaultStateDto.ON)
 check(`AliasDefaultDto default`()==null)
 val option=org.example.implicitstringoptions.api.models.AlphaResponseDto()
 check(option.value?.serialName=="on")
 check(org.example.implicitstringoptions.api.models.BetaResponseDto.serializer().descriptor.serialName==org.example.implicitstringoptions.models.ResponseDto.serializer().descriptor.serialName)
 check(FullStateDto.entries.size==239 && FullStateDto.fromSerialName("v238").serialName=="v238")
 val leaf=LeafDto(kind=RootDtoKind.FIXED)
 check(!LeafDto.serializer().descriptor.getElementDescriptor(0).isNullable)
 check(Json.decodeFromString(RootDto.serializer(),Json.encodeToString(RootDto.serializer(),leaf)) is LeafDto)
 check(Json.decodeFromString(RootDto.serializer(),"{\\"kind\\":null}") is UnknownRootDto)
 val wide=WideDto('''+args+''')
 check(WideDto.serializer().descriptor.elementsCount==240 && wide.p239==null)
 val deep="[".repeat(15)+"null"+"]".repeat(15)
 for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
  val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
  check(codec.decodeFromString(WideDto.serializer(),codec.encodeToString(WideDto.serializer(),wide)).p239==null)
  check(codec.encodeToString(serializer<DeepDto>(),codec.decodeFromString<DeepDto>(deep))==deep)
 }
}
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(main)
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runImplicitStringEnumNullContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  validation=(); [[ "$checked" == false ]] || validation+=(--validation)
  for entry in string-enum-null-members:stringenumnullmembers:DefaultApi referenced-string-enum-null-member-defaults:refnullstrings:RefNullStrings nested-string-enum-collection-defaults:nestedstringenum:NestedStringEnum; do
   IFS=: read -r fixture package client <<< "$entry"
   for mode in base optimized; do
    output="$project/build/generated/$fixture";suffix="";options=()
    if [[ "$mode" == optimized ]]; then
     output="$project/build/generated-limits/$fixture";suffix=optimized
     options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
    fi
    args=(--input "$project/$fixture.json" --output "$output" --package-name "org.example.$package$suffix" --api-client-name "$client" --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}"
    "$workspace/openapi-kt" generate "${args[@]}" "${validation[@]}" "${options[@]}" --check
   done
  done
  if [[ "$checked" == true ]]; then
   for entry in dto-allof-unique:dtoallofunique:DefaultClient:true dto-collection-reference-leaves-unique:dtocollectionreferenceleavesunique:DefaultClient:true dto-string-collection-reference-unique:dtostringcollectionreferenceunique:DefaultClient:true nested-object-defaults:nestedobject:NestedObject:false string-enum-collection-bounds:stringenumbounds:StringEnumBounds:true string-enum-collection-defaults:stringenumcollections:StringEnumCollections:false string-enum-unique:enumunique:StringEnumUniqueApiClient:true tuple-enum-references:tupleenums:TupleEnumsApiClient:false; do
    IFS=: read -r fixture package client fixture_checked <<< "$entry"
    fixture_validation=(); [[ "$fixture_checked" == false ]] || fixture_validation+=(--validation)
    args=(--input "$project/$fixture.json" --output "$project/build/generated-limits/$fixture" --package-name "org.example.$package" --api-client-name "$client" --model-name-suffix Dto --model-style "$style")
    "$workspace/openapi-kt" generate "${args[@]}" "${fixture_validation[@]}"
    "$workspace/openapi-kt" generate "${args[@]}" "${fixture_validation[@]}" --check
   done
  fi
  "$workspace/openapi-kt" generate --input "$project/resources.json" --output "$project/build/generated-options" --package-name org.example.implicitstringresources --model-name-suffix Dto --model-style "$style" "${validation[@]}"
  option_args=(--input "$project/options.json" --output "$project/build/generated-limits/options" --package-name org.example.implicitstringoptions --model-name-suffix Dto --model-style "$style" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
  "$workspace/openapi-kt" generate "${option_args[@]}" "${validation[@]}"
  "$workspace/openapi-kt" generate "${option_args[@]}" "${validation[@]}" --check
  python3 - "$project" <<'PY'
from pathlib import Path
import json,sys
p=Path(sys.argv[1])
assert json.loads((p/'build/generated-limits/options/.openapi-kt-names.json').read_text())['entries']
PY
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runImplicitStringEnumNullContracts
  echo "FRESH IMPLICIT STRING ENUM NULL PROFILE $style-$checked PASS: $project"
 done
done
echo "Fresh implicit String enum null projects: $workspace"
