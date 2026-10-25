#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/null-const-object-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullConstObjectDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1])
source=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullConstObjectDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'NullConstObjectDefaultsContracts.kt').write_text(source.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.nullconstdefaults','org.example.nullconstdefaultsoptimized'))
props={'id':{'type':'string'}}
props.update({'p'+str(i):{'const':None} for i in range(239)})
wide={'openapi':'3.1.0','components':{'schemas':{'Holder':{'properties':{'value':{'properties':props,'required':['id'],'default':{'id':'wide'}}}}}}}
(p/'wide.json').write_text(json.dumps(wide,indent=2)+'\n')
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { val wide=org.example.nullconstdefaultswide.models.HolderDto(); check(wide.value?.id=="wide" && wide.value?.p238===kotlinx.serialization.json.JsonNull); nullConstObjectDefaultsContracts(); org.example.optimizedcontracts.nullConstObjectDefaultsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runNullConstObjectDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
  flags=(--model-style "$style")
  [[ "$checked" == off ]] || flags+=(--validation)
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-const-object-defaults.json --output "$project/build/generated" --package-name org.example.nullconstdefaults --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-const-object-defaults.json --output "$project/build/generated" --package-name org.example.nullconstdefaults --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-const-object-defaults.json --output "$project/build/generated-limits" --package-name org.example.nullconstdefaultsoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-const-object-defaults.json --output "$project/build/generated-limits" --package-name org.example.nullconstdefaultsoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names --check
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.nullconstdefaultswide --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.nullconstdefaultswide --model-name-suffix Dto "${flags[@]}" --check
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNullConstObjectDefaultsContracts
  echo "FRESH NULL CONST OBJECT DEFAULTS PROFILE $style-$checked PASS: $project"
 done
done
echo "Fresh nullconst object default projects: $workspace"
