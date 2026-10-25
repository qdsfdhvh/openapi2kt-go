#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/null-component-factories-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
 for checked in off on; do
  project="$workspace/$style-$checked"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullComponentFactoriesContracts.kt "$project/src/main/kotlin/org/example/contracts/"
  python3 - "$project" <<'PYINPUT'
from pathlib import Path
import json,sys
p=Path(sys.argv[1])
source=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/NullComponentFactoriesContracts.kt').read_text()
opt=p/'src/main/kotlin/org/example/optimizedcontracts';opt.mkdir(parents=True,exist_ok=True)
(opt/'NullComponentFactoriesContracts.kt').write_text(source.replace('package org.example.contracts','package org.example.optimizedcontracts').replace('org.example.nullcomponentfactories','org.example.nullcomponentfactoriesoptimized'))
props={'p'+str(i):{'type':'array','items':{'type':'null'},'default':[None]} for i in range(240)}
wide={'openapi':'3.1.0','components':{'schemas':{'Holder':{'properties':props}}}}
(p/'wide.json').write_text(json.dumps(wide,indent=2)+'\n')
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { val wide=org.example.nullcomponentfactorieswide.models.HolderDto(); check(wide.p239==listOf(kotlinx.serialization.json.JsonNull)); nullComponentFactoriesContracts(); org.example.optimizedcontracts.nullComponentFactoriesContracts() }\n')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runNullComponentFactoriesContracts") {
 dependsOn(tasks.named("classes"))
 classpath = sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYINPUT
  flags=(--model-style "$style")
  [[ "$checked" == off ]] || flags+=(--validation)
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-component-factories.json --output "$project/build/generated" --package-name org.example.nullcomponentfactories --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-component-factories.json --output "$project/build/generated" --package-name org.example.nullcomponentfactories --model-name-suffix Dto "${flags[@]}" --check
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-component-factories.json --output "$project/build/generated-limits" --package-name org.example.nullcomponentfactoriesoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names
  "$workspace/openapi-kt" generate --input tests/fixtures/http/null-component-factories.json --output "$project/build/generated-limits" --package-name org.example.nullcomponentfactoriesoptimized --model-name-suffix Dto "${flags[@]}" --deduplicate-models --hoist-shared-shapes --merge-identical-models --lock-hoisted-names --check
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.nullcomponentfactorieswide --model-name-suffix Dto "${flags[@]}"
  "$workspace/openapi-kt" generate --input "$project/wide.json" --output "$project/build/generated-options" --package-name org.example.nullcomponentfactorieswide --model-name-suffix Dto "${flags[@]}" --check
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runNullComponentFactoriesContracts
  echo "FRESH NULL COMPONENT FACTORIES PROFILE $style-$checked PASS: $project"
 done
done
echo "Fresh nullvalue default projects: $workspace"
