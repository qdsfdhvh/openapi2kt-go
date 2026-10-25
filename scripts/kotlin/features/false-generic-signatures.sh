#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-generic-signatures-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for mode in base optimized; do
 project="$workspace/$mode"
 mkdir -p "$project/src/main/kotlin/org/example/contracts"
 cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
 python3 - "$project" "$mode" "$workspace/openapi-kt" <<'PYPROJECT'
from pathlib import Path
import json,subprocess,sys
p=Path(sys.argv[1]);mode=sys.argv[2];binary=sys.argv[3];tail='example.'+'.'.join(['x'*60]*5)
options=[] if mode=='base' else ['--deduplicate-models','--hoist-shared-shapes','--merge-identical-models','--lock-hoisted-names']
bindings=[];imports=[];constructs=[]
for style,count,prefix,out,alias,required in [
 ('data_class',179,'org' if mode=='base' else 'opt','generated','DataWide',False),
 ('plain_class',180,'com' if mode=='base' else 'alt','generated-limits','PlainWide',False),
 ('data_class',179,'req' if mode=='base' else 'rqd','generated-required','RequiredData',True),
 ('plain_class',180,'rpl' if mode=='base' else 'rqp','generated-required-plain','RequiredPlain',True)]:
 package=prefix+'.'+tail
 props={f'p{i:03d}':{'type':'array','items':False} for i in range(count)};schema={'properties':props}
 if required:schema['required']=list(props)
 input=p/(alias+'.json');input.write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'Wide':schema}}})+'\n');output=p/'build'/out
 args=['generate','--input',str(input),'--output',str(output),'--package-name',package,'--model-name-suffix','Dto','--model-style',style,*options]
 subprocess.run([binary,*args],check=True);subprocess.run([binary,*args,'--check'],check=True)
 records={'style':style,'count':count,'required':required,'input':str(input),'output':str(output),'package':package,'options':options};bindings.append(records)
 imports.append('import '+package+'.models.WideDto as '+alias)
 if required:constructs.append('private fun construct'+alias+'() = '+alias+'('+','.join(['emptyList()']*count)+')')
 bad=p/(alias+'-bad.json');badprops={f'p{i:03d}':{'type':'array','items':False} for i in range(count+1)};badschema={'properties':badprops}
 if required:badschema['required']=list(badprops)
 bad.write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'Wide':badschema}}})+'\n')
 failed=subprocess.run([binary,'generate','--input',str(bad),'--output',str(output),'--check','--package-name',package,'--model-name-suffix','Dto','--model-style',style,'--diagnostic-format=json',*options],capture_output=True,text=True)
 report=json.loads(failed.stdout);assert failed.returncode==1 and not failed.stderr and report['diagnostics'][0]['code']=='E_MODEL_LIMIT' and report['diagnostics'][0]['pointer']=='/components/schemas/Wide',report
(p/'producer-bindings.json').write_text(json.dumps(bindings,indent=2)+'\n')
source='''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
'''
source+='\n'.join(imports+constructs)+'\n'
source+='''fun main() {
 val data=DataWide()
 val plain=PlainWide()
 check(data.copy()==data)
 val requiredData=constructRequiredData()
 val requiredPlain=constructRequiredPlain()
 check(!RequiredData.serializer().descriptor.isElementOptional(0))
 check(!RequiredPlain.serializer().descriptor.isElementOptional(0))
 for(defaults in listOf(false,true)) for(nulls in listOf(false,true)) {
  val codec=Json {encodeDefaults=defaults;explicitNulls=nulls}
  check(codec.parseToJsonElement(codec.encodeToString(data)).jsonObject.size==if(defaults&&nulls) 179 else 0)
  check(codec.parseToJsonElement(codec.encodeToString(plain)).jsonObject.size==if(defaults&&nulls) 180 else 0)
  check(codec.decodeFromString<DataWide>(codec.encodeToString(data)).p178==null)
  check(codec.decodeFromString<PlainWide>(codec.encodeToString(plain)).p179==null)
  check(runCatching {codec.decodeFromString<DataWide>("{\\"p178\\":[null]}")}.isFailure)
  check(runCatching {codec.decodeFromString<PlainWide>("{\\"p179\\":[null]}")}.isFailure)
  val dataBody=codec.encodeToString(requiredData)
  val plainBody=codec.encodeToString(requiredPlain)
  check(codec.parseToJsonElement(dataBody).jsonObject.size==179)
  check(codec.parseToJsonElement(plainBody).jsonObject.size==180)
  check(codec.decodeFromString<RequiredData>(dataBody).p178.isEmpty())
  check(codec.decodeFromString<RequiredPlain>(plainBody).p179.isEmpty())
 }
}
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(source)
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
kotlin.sourceSets.main {
 kotlin.srcDir(layout.buildDirectory.dir("generated-required"))
 kotlin.srcDir(layout.buildDirectory.dir("generated-required-plain"))
}
tasks.register<JavaExec>("runFalseGenericSignatureContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYPROJECT
 if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
  echo "PREPARED FALSE GENERIC SIGNATURES $mode: $project"
  continue
 fi
 "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -Pkotlin.compiler.execution.strategy=in-process -p "$project" runFalseGenericSignatureContracts
 echo "FRESH FALSE GENERIC SIGNATURES PROFILE $mode PASS: $project"
done
