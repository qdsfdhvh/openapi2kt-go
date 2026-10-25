#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/tuple-enum-references-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts" "$project/src/main/kotlin/org/example/optimizedcontracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/TupleEnumReferencesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$style" "$checked" <<'PY'
import json,sys
from pathlib import Path
p=Path(sys.argv[1]);plain=sys.argv[2]=='plain_class';checked=sys.argv[3]=='on'
doc=json.loads(Path('tests/fixtures/http/tuple-enum-references.json').read_text())
if checked:doc['components']['schemas']['Rows'].update(minItems=1,maxItems=4)
doc['components']['schemas']['Wide']={'properties':{f'v{i}':{'$ref':'#/components/schemas/Rows'} for i in range(240)},'required':[f'v{i}' for i in range(240)]}
(p/'input.json').write_text(json.dumps(doc)+'\n')
consumer=(p/'src/main/kotlin/org/example/contracts/TupleEnumReferencesContracts.kt').read_text()
(p/'src/main/kotlin/org/example/optimizedcontracts/TupleEnumReferencesContracts.kt').write_text(consumer.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.tupleenums','org.example.tupleenumsoptimized'))
extra='' if plain else 'check(value.copy(rows=listOf(org.example.tupleenums.models.ChoiceDto.DONE)).rows.single()==org.example.tupleenums.models.ChoiceDto.DONE)'
arguments=','.join(f'v{i}=rows' for i in range(240))
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main()=runBlocking {
    val rows=listOf(org.example.tupleenums.models.ChoiceDto.READY)
    val value=org.example.tupleenums.models.HolderDto(rows,emptyList(),emptyList(),rows,emptyList())
    '''+extra+'''
    check(org.example.tupleenums.models.WideDto('''+arguments+''').v239==rows)
    tupleEnumReferencesContracts('''+str(checked).lower()+''')
    org.example.optimizedcontracts.tupleEnumReferencesContracts('''+str(checked).lower()+''')
}
''')
build=p/'build.gradle.kts';build.write_text(build.read_text()+'''
tasks.register<JavaExec>("runTupleEnumReferencesContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.MainKt")
}
''')
PY
  flags=(--model-style "$style");if [[ "$checked" == on ]];then flags+=(--validation);fi
  for optimized in no yes; do
   pkg=org.example.tupleenums;options=()
   if [[ "$optimized" == yes ]];then pkg=org.example.tupleenumsoptimized;options=(--deduplicate-models --merge-identical-models --hoist-shared-shapes --lock-hoisted-names);fi
   for action in generate check; do
    check=();if [[ "$action" == check ]];then check=(--check);fi
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated/$optimized" --package-name "$pkg" --model-name-suffix Dto "${flags[@]}" "${options[@]}" "${check[@]}"
   done
  done
  if [[ "${OPENAPI_KT_PREPARE_ONLY:-0}" == 1 ]];then
   printf 'PREPARED TUPLE ENUM REFERENCES %s %s: %s\n' "$style" "$checked" "$project"
  else
   "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runTupleEnumReferencesContracts
   printf 'FRESH TUPLE ENUM REFERENCES PROFILE %s %s PASS: %s\n' "$style" "$checked" "$project"
  fi
 done
done
printf 'Fresh tuple enum reference projects: %s\n' "$workspace"
