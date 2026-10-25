#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/jvm-copy-signatures-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for mode in base optimized; do
 project="$workspace/$mode"
 mkdir -p "$project/src/main/kotlin/org/example/contracts"
 cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
 python3 - "$project" "$mode" "$workspace/openapi-kt" <<'PYPROJECT'
from pathlib import Path
import sys,json,subprocess
p=Path(sys.argv[1]);mode=sys.argv[2];binary=sys.argv[3]
leaf={'type':'integer','minimum':1}
for _ in range(16):leaf={'type':'array','items':leaf}
input=p/'input.json';input.write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'Wide':{'properties':{'p'+str(i):leaf for i in range(203)}}}}})+'\n')
plainPackage='org.example.'+'.'.join(['x'*60]*5)
dataPackage='org.example.copylimits'
options=[]
if mode=='optimized':
 options=['--deduplicate-models','--hoist-shared-shapes','--merge-identical-models','--lock-hoisted-names'];plainPackage+='.opt';dataPackage+='optimized'
records=[]
for style,package,output in [('data_class',dataPackage,p/'build/generated'),('plain_class',plainPackage,p/'build/generated-limits')]:
 args=['generate','--input',str(input),'--output',str(output),'--package-name',package,'--model-name-suffix','Dto','--model-style',style,'--validation',*options]
 subprocess.run([binary,*args],check=True);subprocess.run([binary,*args,'--check'],check=True)
 records.append({'style':style,'package':package,'output':str(output),'options':options})
run=subprocess.run([binary,'validate','--input',str(input),'--package-name',plainPackage,'--model-style','data_class','--validation','--diagnostic-format=json'],capture_output=True,text=True)
assert run.returncode==1 and not run.stderr,(run.stdout,run.stderr)
report=json.loads(run.stdout);assert report['diagnostics'][0]['code']=='E_MODEL_LIMIT' and report['diagnostics'][0]['pointer']=='/components/schemas/Wide',report
(p/'producer-bindings.json').write_text(json.dumps(records,indent=2)+'\n')
valid='1';invalid='0'
for _ in range(16):valid='listOf('+valid+')';invalid='listOf('+invalid+')'
source="""@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
package org.example.contracts
import kotlinx.serialization.json.Json
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
"""
source+='import '+dataPackage+'.models.WideDto as DataWide\nimport '+plainPackage+'.models.WideDto as PlainWide\n'
source+='fun main() {\nval data=DataWide(p0='+valid+')\nval plain=PlainWide(p0='+valid+')\ncheck(data.copy()==data)\n'
for name in ['DataWide','PlainWide']:
 source+='check(runCatching { '+name+'(p202='+invalid+') }.isFailure)\n'
 source+='for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) { val codec=Json {encodeDefaults=defaults;explicitNulls=nulls};check(codec.decodeFromString<'+name+'>(codec.encodeToString('+('data' if name=='DataWide' else 'plain')+')) == '+('data' if name=='DataWide' else 'plain')+') }\n'
source+='}\n';(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text(source)
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runJVMCopySignatureContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYPROJECT
 if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
  echo "PREPARED JVM COPY SIGNATURES $mode: $project"
  continue
 fi
 "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -Pkotlin.compiler.execution.strategy=in-process -p "$project" runJVMCopySignatureContracts
 echo "FRESH JVM COPY SIGNATURES PROFILE $mode PASS: $project"
done
