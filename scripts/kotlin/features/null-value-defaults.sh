#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/null-value-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullValueDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1])
source=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullValueDefaultsContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'NullValueDefaultsContracts.kt').write_text(source.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.nullvaluedefaults','org.example.nullvaluedefaultsoptimized'))
props={'p'+str(i):{'type':'array','items':{'type':'null'},'default':[None]} for i in range(240)}
wide={'openapi':'3.1.0','components':{'schemas':{'Holder':{'properties':props}}}}
(p/'wide.json').write_text(json.dumps(wide,indent=2)+'\n')
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { val wide=org.example.nullvaluedefaultswide.models.HolderDto(); check(wide.p239==listOf(kotlinx.serialization.json.JsonNull)); nullValueDefaultsContracts(); org.example.optimizedcontracts.nullValueDefaultsContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runNullValueDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
  flags=(--model-style "$style")
  [[ "$checked" == off ]] || flags+=(--validation)
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-value-defaults.json --output "$project/build/generated" --package-name org.example.nullvaluedefaults --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-value-defaults.json --output "$project/build/generated" --package-name org.example.nullvaluedefaults --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-value-defaults.json --output "$project/build/generated-limits" --package-name org.example.nullvaluedefaultsoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-value-defaults.json --output "$project/build/generated-limits" --package-name org.example.nullvaluedefaultsoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names --check
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.nullvaluedefaultswide --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.nullvaluedefaultswide --model-name-suffix Dto "${flags[@]}" --check
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNullValueDefaultsContracts
  echo "FRESH NULL VALUE DEFAULTS PROFILE $style-$checked PASS: $project"
 done
done
echo "Fresh nullvalue default projects: $workspace"
