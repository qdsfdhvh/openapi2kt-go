#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-target-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts" "$project/src/main/kotlin/org/example/optimizedcontracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleTargetDefaultsReferencesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$style" "$checked" <<'PY'
import json,sys
from pathlib import Path
p=Path(sys.argv[1]);plain=sys.argv[2]=='plain_class';checked=sys.argv[3]=='on'
doc=json.loads(Path('tests/fixtures/http/tuple-target-defaults.json').read_text())
s=doc['components']['schemas']
if checked:
 s['Rows'].update(minItems=1,maxItems=3)
 # Own counts on composed roots retain their existing rejection; constructor
 # member/leaf checks use the original allOf source contributions.
 s['Base']['properties']['value']['minLength']=2
 for name in ('Item','Other'):
  part=s[name]['allOf'][1]['properties']
  part['count']['minimum']=0
  part['child']['properties']['label']['minLength']=2
 s['Other']['allOf'][0]['properties']['value']['minLength']=2
 s['Node']['allOf'][0]['properties']['value']['minLength']=2
 s['Owner']['properties']['node']['allOf'][0]['properties']['label']['minLength']=2
s['Wide']={'properties':{f'v{i}':{'$ref':'#/components/schemas/Rows'} for i in range(240)},'required':[f'v{i}' for i in range(240)]}
(p/'input.json').write_text(json.dumps(doc)+'\n')
consumer=(p/'src/main/kotlin/org/example/contracts/TupleTargetDefaultsReferencesContracts.kt').read_text()
(p/'src/main/kotlin/org/example/optimizedcontracts/TupleTargetDefaultsReferencesContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.tupletargetdefaults','org.example.tupletargetdefaultsoptimized'))
extra='' if plain else '''val item=org.example.tupletargetdefaults.models.ItemDto(value="valid")
    check(item.copy(value="copy").value=="copy")
    '''+('check(runCatching { item.copy(count=-1) }.isFailure)' if checked else '')
arguments=','.join(f'v{i}=rows' for i in range(240))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main()=runBlocking {
    val rows=listOf(org.example.tupletargetdefaults.models.ItemDto(value="valid"))
    '''+extra+'''
    check(org.example.tupletargetdefaults.models.WideDto('''+arguments+''').v239==rows)
    tupleTargetDefaultsReferencesContracts('''+str(checked).lower()+''')
    org.example.optimizedcontracts.tupleTargetDefaultsReferencesContracts('''+str(checked).lower()+''')
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runTupleTargetDefaultsReferencesContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  flags=(--model-style "$style" --api-client-name TupleTargetDefaultsClient);if [[ "$checked" == on ]];then flags+=(--validation);fi
  for optimized in no yes; do
   pkg=org.example.tupletargetdefaults;options=()
   if [[ "$optimized" == yes ]];then pkg=org.example.tupletargetdefaultsoptimized;options=(--deduplicate-models --merge-identical-models --hoist-shared-shapes --lock-hoisted-names);fi
   for action in generate check; do
    check=();if [[ "$action" == check ]];then check=(--check);fi
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated/$optimized" --package-name "$pkg" --model-name-suffix Dto "${flags[@]}" "${options[@]}" "${check[@]}"
   done
   if [[ "$optimized" == yes ]];then
    python3 - "$project/build/generated/$optimized/.openapi-kt-names.json" <<'PY'
import json,sys
assert json.load(open(sys.argv[1]))['entries']
PY
   fi
  done
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]];then
   printf 'PREPARED TUPLE TARGET DEFAULTS REFERENCES %s %s: %s\n' "$style" "$checked" "$project"
  else
   "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleTargetDefaultsReferencesContracts
   printf 'FRESH TUPLE TARGET DEFAULTS REFERENCES PROFILE %s %s PASS: %s\n' "$style" "$checked" "$project"
  fi
 done
done
printf 'Fresh tuple target default projects: %s\n' "$workspace"
