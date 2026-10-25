#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-dto-references-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts" "$project/src/main/kotlin/org/example/optimizedcontracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleDTOReferencesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$style" "$checked" <<'PY'
import json,sys
from pathlib import Path
p=Path(sys.argv[1]);plain=sys.argv[2]=='plain_class';checked=sys.argv[3]=='on'
doc=json.loads(Path('tests/fixtures/http/tuple-dto-references.json').read_text())
s=doc['components']['schemas']
if checked:
 s['Rows'].update(minItems=1,maxItems=3)
 for name in ('Item','Other'):
  s[name].update(minProperties=1,maxProperties=4)
  s[name]['properties']['value']['minLength']=2
  s[name]['properties']['count']['minimum']=0
  s[name]['properties']['child']['properties']['label']['minLength']=2
 s['Node']['properties']['value']['minLength']=2
 s['Owner']['properties']['node']['properties']['label']['minLength']=2
s['Wide']={'properties':{f'v{i}':{'$ref':'#/components/schemas/Rows'} for i in range(240)},'required':[f'v{i}' for i in range(240)]}
(p/'input.json').write_text(json.dumps(doc)+'\n')
consumer=(p/'src/main/kotlin/org/example/contracts/TupleDTOReferencesContracts.kt').read_text()
(p/'src/main/kotlin/org/example/optimizedcontracts/TupleDTOReferencesContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.tupledtos','org.example.tupledtosoptimized'))
extra='' if plain else '''val item=org.example.tupledtos.models.ItemDto(value="valid")
    check(item.copy(value="copy").value=="copy")
    '''+('check(runCatching { item.copy(count=-1) }.isFailure)' if checked else '')
arguments=','.join(f'v{i}=rows' for i in range(240))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main()=runBlocking {
    val rows=listOf(org.example.tupledtos.models.ItemDto(value="valid"))
    '''+extra+'''
    check(org.example.tupledtos.models.WideDto('''+arguments+''').v239==rows)
    tupleDTOReferencesContracts('''+str(checked).lower()+''')
    org.example.optimizedcontracts.tupleDTOReferencesContracts('''+str(checked).lower()+''')
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runTupleDTOReferencesContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  flags=(--model-style "$style" --api-client-name TupleDTOClient);if [[ "$checked" == on ]];then flags+=(--validation);fi
  for optimized in no yes; do
   pkg=org.example.tupledtos;options=()
   if [[ "$optimized" == yes ]];then pkg=org.example.tupledtosoptimized;options=(--deduplicate-models --merge-identical-models --hoist-shared-shapes --lock-hoisted-names);fi
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
   printf 'PREPARED TUPLE DTO REFERENCES %s %s: %s\n' "$style" "$checked" "$project"
  else
   "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleDTOReferencesContracts
   printf 'FRESH TUPLE DTO REFERENCES PROFILE %s %s PASS: %s\n' "$style" "$checked" "$project"
  fi
 done
done
printf 'Fresh tuple DTO reference projects: %s\n' "$workspace"
