#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/false-nominal-signatures-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for mode in base optimized; do
 project="$workspace/$mode"
 mkdir -p "$project/src/main/kotlin/org/example/contracts"
 cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
 python3 - "$project" "$mode" "$workspace/openapi-kt" <<'PYPROJECT'
from pathlib import Path
import json,subprocess,sys
p=Path(sys.argv[1]);mode=sys.argv[2];binary=sys.argv[3];tail='example.'+'.'.join(['x'*60]*5)
dataPackage=('org' if mode=='base' else 'opt')+'.'+tail
plainPackage=('com' if mode=='base' else 'alt')+'.'+tail
options=[] if mode=='base' else ['--deduplicate-models','--hoist-shared-shapes','--merge-identical-models','--lock-hoisted-names']
records=[]
for style,count,package,output in [('data_class',188,dataPackage,p/'build/generated'),('plain_class',190,plainPackage,p/'build/generated-limits')]:
 input=p/(style+'.json');input.write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'Wide':{'properties':{f'p{i:03d}':False for i in range(count)}}}}})+'\n')
 args=['generate','--input',str(input),'--output',str(output),'--package-name',package,'--model-name-suffix','Dto','--model-style',style,*options]
 subprocess.run([binary,*args],check=True);subprocess.run([binary,*args,'--check'],check=True)
 for bad_count in [count+1,240]:
  bad=p/(style+'-'+str(bad_count)+'.json');bad.write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'Wide':{'properties':{f'p{i:03d}':False for i in range(bad_count)}}}}})+'\n')
  failed=subprocess.run([binary,'generate','--input',str(bad),'--output',str(output),'--check','--package-name',package,'--model-name-suffix','Dto','--model-style',style,'--diagnostic-format=json',*options],capture_output=True,text=True)
  report=json.loads(failed.stdout);assert failed.returncode==1 and not failed.stderr and report['diagnostics'][0]['code']=='E_MODEL_LIMIT' and report['diagnostics'][0]['pointer']=='/components/schemas/Wide',report
 records.append({'style':style,'count':count,'package':package,'input':str(input),'output':str(output),'options':options})
(p/'producer-bindings.json').write_text(json.dumps(records,indent=2)+'\n')
source='''@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.serialization.json.Json
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
'''
source+='import '+dataPackage+'.models.WideDto as DataWide\nimport '+plainPackage+'.models.WideDto as PlainWide\n'
source+='fun main() {\nval data=DataWide()\nval plain=PlainWide()\ncheck(data.copy()==data)\n'
for name,count in [('DataWide',188),('PlainWide',190)]:
 source+='check('+name+'.serializer().descriptor.elementsCount=='+str(count)+')\n'
 source+='for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) {\nval codec=Json { encodeDefaults=defaults; explicitNulls=nulls }\n'
 source+='check(codec.encodeToString('+('data' if name=='DataWide' else 'plain')+')=="{}")\n'
 source+='check(codec.encodeToString(codec.decodeFromString<'+name+'>("{}"))=="{}")\n'
 source+='check(runCatching {codec.decodeFromString<'+name+'>("{\\"p'+str(count-1).zfill(3)+'\\":null}")}.isFailure)\n}\n'
source+='}\n';(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(source)
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runFalseNominalSignatureContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYPROJECT
 if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
  echo "PREPARED FALSE NOMINAL SIGNATURES $mode: $project"
  continue
 fi
 "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -Pkotlin.compiler.execution.strategy=in-process -p "$project" runFalseNominalSignatureContracts
 echo "FRESH FALSE NOMINAL SIGNATURES PROFILE $mode PASS: $project"
done
