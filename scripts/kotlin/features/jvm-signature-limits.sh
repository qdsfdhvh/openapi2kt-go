#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/jvm-signature-limits-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 project="$workspace/$style"
 mkdir -p "$project/src/main/kotlin/org/example/contracts"
 cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
 python3 - "$project" "$style" <<'PYSOURCE'
from pathlib import Path
import sys,json
p=Path(sys.argv[1]);style=sys.argv[2]
leaf={'type':'integer','minimum':1}
for _ in range(16):leaf={'type':'array','items':leaf}
for width in [203,204]:
 (p/(str(width)+'.json')).write_text(json.dumps({'openapi':'3.1.0','components':{'schemas':{'Wide':{'properties':{'p'+str(i):leaf for i in range(width)}}}}})+'\n')
valid='1';invalid='0'
for _ in range(16):valid='listOf('+valid+')';invalid='listOf('+invalid+')'
source='package org.example.contracts\nimport kotlinx.serialization.json.Json\nimport kotlinx.serialization.encodeToString\nimport kotlinx.serialization.decodeFromString\nfun main() {\n'
for package in ['org.example.jvmlimits','org.example.jvmlimitsoptimized']:
 source+='run {\nval value='+package+'.models.WideDto(p0='+valid+')\n'
 if style=='data_class':source+='check(value.copy()==value)\n'
 source+='check(runCatching { '+package+'.models.WideDto(p202='+invalid+') }.isFailure)\n'
 source+='for (defaults in listOf(false,true)) for (nulls in listOf(false,true)) { val codec=Json {encodeDefaults=defaults;explicitNulls=nulls};check(codec.decodeFromString<'+package+'.models.WideDto>(codec.encodeToString(value))==value) }\n}\n'
source+='}\n'
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('@file:OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)\n'+source)
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runJVMSignatureLimitsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYSOURCE
 for mode in base optimized; do
  package=org.example.jvmlimits;output="$project/build/generated";options=()
  if [[ "$mode" == optimized ]]; then
   package=org.example.jvmlimitsoptimized;output="$project/build/generated-limits"
   options+=(--deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names)
  fi
  args=(--input "$project/203.json" --output "$output" --package-name "$package" --model-name-suffix Dto --model-style "$style" --validation)
  "$workspace/openapi-kt" generate "${args[@]}" "${options[@]}"
  "$workspace/openapi-kt" generate "${args[@]}" "${options[@]}" --check
  python3 - "$workspace/openapi-kt" "$project/204.json" "$output" "$package" "$style" "${options[@]}" <<'PYREJECT'
from pathlib import Path
import subprocess,sys,json,hashlib
binary,input,output,package,style,*options=sys.argv[1:]
def inventory():return {str(p.relative_to(output)):hashlib.sha256(p.read_bytes()).hexdigest() for p in Path(output).rglob('*') if p.is_file()}
before=inventory()
run=subprocess.run([binary,'generate','--input',input,'--output',output,'--package-name',package,'--model-name-suffix','Dto','--model-style',style,'--validation','--diagnostic-format=json',*options],capture_output=True,text=True)
assert run.returncode==1 and not run.stderr,(run.returncode,run.stdout,run.stderr)
report=json.loads(run.stdout);assert report['diagnostics'][0]['code']=='E_MODEL_LIMIT' and report['diagnostics'][0]['pointer']=='/components/schemas/Wide',report
assert before==inventory(),'overflow changed old output'
PYREJECT
 done
 if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]]; then
  echo "PREPARED JVM SIGNATURE LIMITS $style: $project"
  continue
 fi
 "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -Pkotlin.compiler.execution.strategy=in-process -p "$project" runJVMSignatureLimitsContracts
 echo "FRESH JVM SIGNATURE LIMITS PROFILE $style PASS: $project"
done
