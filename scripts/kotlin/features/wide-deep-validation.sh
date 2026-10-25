#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/wide-deep-validation-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  project="$workspace/$style"
  mkdir -p "$project/src/main/kotlin/org/example/contracts"
  cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
  for mode in base dedup hoist merge dedup_hoist dedup_merge hoist_merge all; do
    options=(); package="org.example.widedeep$mode"
    [[ "$mode" != base ]] || package=org.example.widedeep
    case "$mode" in dedup|dedup_hoist|dedup_merge|all) options+=(--deduplicate-models);; esac
    case "$mode" in hoist|dedup_hoist|hoist_merge|all) options+=(--hoist-shared-shapes --lock-hoisted-names);; esac
    case "$mode" in merge|dedup_merge|hoist_merge|all) options+=(--merge-identical-models);; esac
    output="$project/build/generated-options/$mode"
    args=(--input tests/fixtures/http/wide-deep-validation.json --output "$output" --package-name "$package" --model-name-suffix Dto --model-style "$style" --validation)
    "$workspace/openapi-kt" generate "${args[@]}" "${options[@]}"
    "$workspace/openapi-kt" generate "${args[@]}" "${options[@]}" --check
    python3 - "$project" "$mode" "$package" "$style" <<'PYCONSUMER'
from pathlib import Path
import sys
p=Path(sys.argv[1]);mode=sys.argv[2];package=sys.argv[3]
s=Path('samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/WideDeepValidationContracts.kt').read_text()
s=s.replace('org.example.widedeep',package)
if sys.argv[4]=='data_class':
 s=s.replace('    val wide12 = WideTwelveDto.default()', '''    val wide12 = WideTwelveDto.default()
    check(wide16.copy() == wide16)
    check(runCatching {wide16.copy(p199=zero16)}.isFailure)''')
if mode!='base':s=s.replace('WideDeepValidation','WideDeepValidation'+mode).replace('wideDeepValidation','wideDeepValidation'+mode)
(p/'src/main/kotlin/org/example/contracts'/('WideDeepValidation'+mode+'Contracts.kt')).write_text(s)
PYCONSUMER
  done
  python3 - "$project" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1])
modes=['base','dedup','hoist','merge','dedup_hoist','dedup_merge','hoist_merge','all']
calls='; '.join('wideDeepValidation'+('' if mode=='base' else mode)+'Contracts()' for mode in modes)
(p/'src/main/kotlin/org/example/contracts/HttpContracts.kt').write_text('package org.example.contracts\nimport kotlinx.coroutines.runBlocking\nfun main() = runBlocking { '+calls+' }\n')
f=p/'build.gradle.kts';f.write_text(f.read_text()+'''
tasks.register<JavaExec>("runWideDeepValidationContracts") {
    dependsOn(tasks.named("classes"))
    classpath=sourceSets["main"].runtimeClasspath
    mainClass.set("org.example.contracts.HttpContractsKt")
}
''')
PYMAIN
  python3 - "$project" "$root" <<'PYPARTITION'
from pathlib import Path
import importlib.util,sys
project=Path(sys.argv[1]);source=Path(sys.argv[2])/'scripts/kotlin/http_compilations.py'
spec=importlib.util.spec_from_file_location('resource_compilations',source)
module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
# Preserve every width/depth case and source. This profile has unusually long
# generic type signatures; use separate compiler invocations under the existing
# heap, with the same audited inventory partitioner as complete HTTP checks.
module.TARGET_BYTES=128*1024
module.partition(project)
PYPARTITION
  "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -Pkotlin.compiler.execution.strategy=in-process -p "$project" runWideDeepValidationContracts
  echo "FRESH WIDE DEEP VALIDATION PROFILE $style ALL8 OPTIONS PASS: $project"
done
echo "Fresh wide/deep validation projects: $workspace"
