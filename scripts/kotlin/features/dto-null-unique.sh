#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/dto-null-unique-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DTONullUniqueContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" "$checked" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1]);d=json.loads(Path('tests/fixtures/http/dto-null-unique.json').read_text())
if sys.argv[2]=='off':
 for name in ['Holder','NullableHolder','InlineHolder']:d['components']['schemas'][name]['properties']['values'].pop('uniqueItems')
(p/'input.json').write_text(json.dumps(d,indent=2)+'\n')
wide={'openapi':'3.1.0','components':{'schemas':{'Item':{'properties':{'p'+str(i):{'type':'string'} for i in range(240)}},'Holder':{'properties':{'values':{'type':'array','items':{'$ref':'#/components/schemas/Item'}}}}}}}
wide['components']['schemas']['Item']['properties']['p0']={'type':'null','const':None}
if sys.argv[2]=='on':wide['components']['schemas']['Holder']['properties']['values']['uniqueItems']=True
wide['components']['schemas']['MegaHolder']={'properties':{'v'+str(i):json.loads(json.dumps(wide['components']['schemas']['Holder']['properties']['values'])) for i in range(240)}}

(p/'wide.json').write_text(json.dumps(wide,indent=2)+'\n')
source=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/DTONullUniqueContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'DTONullUniqueContracts.kt').write_text(source.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.dtonullunique','org.example.dtonulluniqueoptimized'))

(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { check(org.example.dtonulluniquewide.models.HolderDto(values=listOf(org.example.dtonulluniquewide.models.ItemDto(p239="wide"))).values?.single()?.p239=="wide"); check(org.example.dtonulluniquewide.models.MegaHolderDto(v239=listOf(org.example.dtonulluniquewide.models.ItemDto(p239="wide"))).v239?.single()?.p239=="wide"); dtoNullUniqueContracts('+str(sys.argv[2]=='on').lower()+'); org.example.optimizedcontracts.dtoNullUniqueContracts('+str(sys.argv[2]=='on').lower()+') }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runDTONullUniqueContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
  flags=(--model-style "$style")
  [[ "$checked" == off ]] || flags+=(--validation)
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.dtonullunique --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.dtonullunique --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.dtonulluniqueoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names
  "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated-limits" --package-name org.example.dtonulluniqueoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names --check
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.dtonulluniquewide --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.dtonulluniquewide --model-name-suffix Dto "${flags[@]}" --check
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runDTONullUniqueContracts
  echo "FRESH DTO NULL UNIQUE PROFILE $style-$checked PASS: $project"
 done
done
echo "Fresh DTO null unique projects: $workspace"
