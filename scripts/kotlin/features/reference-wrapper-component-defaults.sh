#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/../common.sh"
mkdir -p build
workspace="$(mktemp -d "$root/build/reference-wrapper-component-defaults-smoke-XXXXXX")"
openapi_kt_build_cli "$workspace/openapi-kt"
for style in data_class plain_class; do
  for validation in off on; do
    project="$workspace/$style-$validation"
    mkdir -p "$project/src/main/kotlin/org/example/contracts"
    cp samples/kotlin-http-smoke/{build.gradle.kts,settings.gradle.kts} "$project/"
    cp samples/kotlin-http-smoke/src/main/kotlin/org/example/contracts/ReferenceWrapperComponentDefaultsContracts.kt "$project/src/main/kotlin/org/example/contracts/"
    cp tests/fixtures/http/reference-wrapper-component-defaults.json "$project/input.json"
    flags=()
    [[ "$validation" == off ]] || flags+=(--validation)
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.refwrapper --api-client-name RefWrapper --model-name-suffix Dto --model-style "$style" "${flags[@]}"
    "$workspace/openapi-kt" generate --input "$project/input.json" --output "$project/build/generated" --package-name org.example.refwrapper --api-client-name RefWrapper --model-name-suffix Dto --model-style "$style" "${flags[@]}" --check
    python3 - "$project" "$style" <<'PYMAIN'
from pathlib import Path
import sys
p=Path(sys.argv[1]);copy='' if sys.argv[2]=='plain_class' else '''
 val value=org.example.refwrapper.models.HolderDto()
 check(value.copy(mode=org.example.refwrapper.models.ModeDto.B).mode==org.example.refwrapper.models.ModeDto.B)
 check(value.copy().rows==null)
 val adopted=value.copy(nested=org.example.refwrapper.models.`WrappedNestedDto default`(), raw=org.example.refwrapper.models.`WrappedRawDto default`())
 check(adopted.nested?.get(0)?.get("a")==2)
 check(adopted.copy(nested=null, raw=null).nested==null)
 check(adopted.copy(nested=null, raw=null).raw==null)
'''
(p/'src/main/kotlin/org/example/contracts/Main.kt').write_text('''package org.example.contracts
import kotlinx.coroutines.runBlocking
fun main() = runBlocking {
'''+copy+''' referenceWrapperComponentDefaultsContracts() }
''')
b=p/'build.gradle.kts';b.write_text(b.read_text()+'''
tasks.register<JavaExec>("runReferenceWrapperComponentDefaultsContracts") {
 dependsOn(tasks.named("classes"))
 classpath=sourceSets["main"].runtimeClasspath
 mainClass.set("org.example.contracts.MainKt")
}
''')
PYMAIN
    "${GRADLE:-gradle}" --offline --no-daemon --no-build-cache --rerun-tasks -p "$project" runReferenceWrapperComponentDefaultsContracts
    echo "FRESH REFERENCE WRAPPER DEFAULTS PROFILE $style-$validation PASS: $project"
  done
done
echo "Fresh reference wrapper default native projects: $workspace"
