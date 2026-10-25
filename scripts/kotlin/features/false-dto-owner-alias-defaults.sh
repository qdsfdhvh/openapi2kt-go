#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-dto-owner-alias-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/FalseDTOOwnerAliasDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  validation=(); [[ "$checked" == off ]] || validation+=(--validation)
  python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);checked=sys.argv[2]=='on'
d=json.loads(Path('tests/fixtures/http/false-dto-owner-alias-defaults.json').read_text())
if checked:d['components']['schemas']['CountedTarget'].update(minProperties=0,maxProperties=1)
(p/'input.json').write_text(json.dumps(d)+'\n')
leaf=False;value=[]
for i in range(15):
 layer={'type':'array','items':leaf}
 if checked:layer['maxItems']=1
 leaf=layer
 if i>0:value=[value]
shallow={'type':'array','items':False}
if checked:shallow['maxItems']=0
ref={'$ref':'#/components/schemas/Alias'};props={f'p{i:03d}':ref for i in range(240)}
values={name:{'rows':[]} for name in props}
schemas={'Alias':{'$ref':'#/components/schemas/Target'},'DeepAlias':{'$ref':'#/components/schemas/DeepTarget'},'Target':{'required':['rows'],'properties':{'rows':shallow},'default':{'rows':[]}},'Wide':{'properties':props,'default':values},'WideRequired':{'required':list(props),'properties':props,'default':values},'DeepTarget':{'required':['rows'],'properties':{'rows':leaf},'default':{'rows':value}},'DeepOwner':{'properties':{'target':{'$ref':'#/components/schemas/DeepAlias'}},'default':{'target':{'rows':value}}}}
if checked:schemas['WideRequired']['minProperties']=240
schemas['LongTarget']={'properties':{'blocked':False}}
last='LongTarget'
for i in range(46):
 name=f'LongAlias{i:02d}';schemas[name]={'$ref':'#/components/schemas/'+last};last=name
schemas['LongOwner']={'properties':{'target':{'$ref':'#/components/schemas/'+last}},'default':{'target':{}}}
(p/'wide.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':schemas}})+'\n')
large={'Alias':{'$ref':'#/components/schemas/Target'},'Target':{'required':['labels'],'properties':{'labels':{'type':'array','items':{'type':'array','nullable':True,'items':False}}},'default':{'labels':[]}},'Owner':{'properties':{'target':{'$ref':'#/components/schemas/Alias'}},'default':{'target':{'labels':[None]*6549}}}}
(p/'large.json').write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':large}})+'\n')
PYINPUT
  for mode in base optimized wide large; do
   input="$project/input.json";output="$project/build/generated";package=org.example.falsedtoaliasdefaults;suffix=;options=();client=(--api-client-name FalseDTOAliasDefaultsClient)
   if [[ "$mode" == optimized ]]; then
    output="$project/build/generated-limits";package+=optimized;suffix=Dto
    options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
   elif [[ "$mode" == wide ]]; then
    input="$project/wide.json";output="$project/build/generated-options";package+=wide;suffix=Dto;client=()
   elif [[ "$mode" == large ]]; then
    input="$project/large.json";output="$project/build/generated-large";package+=large;suffix=Dto;client=()
   fi
   args=(--input "$input" --output "$output" --package-name "$package" --model-name-suffix "$suffix" --model-style "$style")
   "$workspace/openapi-kt" generate "${args[@]}" "${client[@]}" "${validation[@]}" "${options[@]}"
   "$workspace/openapi-kt" generate "${args[@]}" "${client[@]}" "${validation[@]}" "${options[@]}" --check
  done
  python3 - "$project" <<'PYMAIN'
from pathlib import Path
import json,re,sys
p=Path(sys.argv[1]);consumer=(p/'src/main/kotlin/org/example/contracts/FalseDTOOwnerAliasDefaultsContracts.kt').read_text()
optimized=consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.falsedtoaliasdefaults.FalseDTOAliasDefaultsClient','org.example.falsedtoaliasdefaultsoptimized.FalseDTOAliasDefaultsClient')
optimized=re.sub(r'\borg\.example\.falsedtoaliasdefaults\.models\.([A-Za-z0-9_]+)',lambda m:'org.example.falsedtoaliasdefaultsoptimized.models.'+m.group(1)+'Dto',optimized)
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True);(opt/'FalseDTOOwnerAliasDefaultsContracts.kt').write_text(optimized)
assert json.loads((p/'build/generated-limits/.openapi-kt-names.json').read_text())['entries'],'unrelated ordinary requests retain name locks'
wide='''package org.example.contracts
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
fun wideFalseDTOOwnerDefaults() {
 check(org.example.falsedtoaliasdefaultswide.models.LongOwnerDto.default().target!!.blocked==null)
 val first=org.example.falsedtoaliasdefaultswide.models.WideDto.default()
 val second=org.example.falsedtoaliasdefaultswide.models.WideDto.default()
 check(first.p000 !== second.p000 && first.p000 !== first.p001)
 check(first.p000!!.rows.isEmpty() && first.p239!!.rows.isEmpty())
 val required=requiredWide(org.example.falsedtoaliasdefaultswide.models.TargetDto(rows=emptyList()))
 check(required.p000.rows.isEmpty())
 check(org.example.falsedtoaliasdefaultswide.models.WideRequiredDto.default().p239.rows.isEmpty())
 val descriptor=org.example.falsedtoaliasdefaultswide.models.WideRequiredDto.serializer().descriptor
 check(descriptor.elementsCount==240)
 for(index in 0 until 240) check(!descriptor.isElementOptional(index))
 val deep=org.example.falsedtoaliasdefaultswide.models.DeepOwnerDto.default()
 check(deep.target!!.rows.size==1)
 val large=org.example.falsedtoaliasdefaultslarge.models.OwnerDto.default()
 val next=org.example.falsedtoaliasdefaultslarge.models.OwnerDto.default()
 check(large.target!!.labels.size==6549 && large.target !== next.target && large.target!!.labels !== next.target!!.labels)
 check(large.target!!.labels.all {it==null})
 for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
  val codec=kotlinx.serialization.json.Json {encodeDefaults=defaults;explicitNulls=nulls}
  check(codec.decodeFromString<org.example.falsedtoaliasdefaultswide.models.WideDto>(codec.encodeToString(first)).p239!!.rows.isEmpty())
  check(codec.decodeFromString<org.example.falsedtoaliasdefaultswide.models.WideRequiredDto>(codec.encodeToString(required)).p000.rows.isEmpty())
  check(codec.decodeFromString<org.example.falsedtoaliasdefaultswide.models.DeepOwnerDto>(codec.encodeToString(deep)).target!!.rows==deep.target!!.rows)
  check(runCatching {codec.decodeFromString<org.example.falsedtoaliasdefaultswide.models.WideDto>("{\\"p000\\":{\\"rows\\":[null]}}")}.isFailure)
 }
}
'''
wide+='private fun requiredWide(target: org.example.falsedtoaliasdefaultswide.models.TargetDto) = org.example.falsedtoaliasdefaultswide.models.WideRequiredDto('+','.join(['target']*240)+')\n'
(p/'src/main/kotlin/org/example/contracts/Wide.kt').write_text(wide)
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main()=runBlocking {wideFalseDTOOwnerDefaults();falseDTOOwnerAliasDefaultsContracts();org.example.optimizedcontracts.falseDTOOwnerAliasDefaultsContracts()}\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
kotlin.sourceSets.main {kotlin.srcDir(layout.buildDirectory.dir("generated-large"))}
tasks.register<JavaExec>("runFalseDTOOwnerAliasDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
   echo "PREPARED FALSE DTO OWNER ALIAS DEFAULTS $style $checked: $project"
   continue
  fi
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runFalseDTOOwnerAliasDefaultsContracts
  echo "FRESH FALSE DTO OWNER ALIAS DEFAULTS PROFILE $style-$checked PASS: $project"
 done
done
